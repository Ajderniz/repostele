package models

import (
	"errors"
	"log/slog"
	"time"
)

type OrderStatus int
const (
  ORDER_STATUS_UNREVIEWED OrderStatus = iota
  ORDER_STATUS_DENIED
  ORDER_STATUS_CANCELLED
  ORDER_STATUS_ACCEPTED
  ORDER_STATUS_FULFILLED
)

// Status helpers for templates, which can't name the constants.
func (s OrderStatus) IsUnreviewed() bool { return s == ORDER_STATUS_UNREVIEWED }
func (s OrderStatus) IsDenied() bool     { return s == ORDER_STATUS_DENIED }
func (s OrderStatus) IsAccepted() bool   { return s == ORDER_STATUS_ACCEPTED }
func (s OrderStatus) IsFulfilled() bool  { return s == ORDER_STATUS_FULFILLED }

// InReview: waiting on staff (unreviewed or denied). Staff can accept or deny;
// the customer can still edit the reference.
func (s OrderStatus) InReview() bool { return s.IsUnreviewed() || s.IsDenied() }

type ItemIdQuant map[int]int

type Order struct {
  Id      int         `db:"id"       json:"id"`
  User    string      `db:"user"     json:"user"`
  Total   float32     `db:"total"    json:"total"`
  RefNum  string      `db:"ref_num"  json:"ref_num"`
  Time    int64       `db:"time"     json:"time"`
  Status  OrderStatus `db:"status"   json:"status"`
  Updated int64       `db:"updated"  json:"updated"`
  Lines   []OrderLine               `json:"lines"`
}

// OrderLine is one item of a new order, priced when the order is placed.
type OrderLine struct {
  ItemID int     `json:"item_id"`
  Quant  int     `json:"quant"`
  Price  float32 `json:"price"`
}

// Pending: the order exists and is still open (unreviewed, denied or accepted).
// Customers may cancel it; only one may be pending per user.
func (o Order) Pending() bool {
  return o.RefNum != "" &&
    o.Status != ORDER_STATUS_CANCELLED && o.Status != ORDER_STATUS_FULFILLED
}

// Editable: the customer may still change the reference (unreviewed or denied).
func (o Order) Editable() bool {
  return o.RefNum != "" && o.Status.InReview()
}

const (
  _ORDERS        = "orders"
  ORDER_ID       = "id"
  _ORDER_USER    = "user"
  _ORDER_TOTAL   = "total"
  ORDER_REF_NUM  = "ref_num"
  _ORDER_TIME    = "time"
  ORDER_STATUS   = "status"
  ORDER_UPDATED  = "updated"

  _ORDER_ITEMS         = "order_items"
  _ORDER_ITEM_ORDER_ID = "order_id"
  _ORDER_ITEM_ITEM_ID  = "item_id"
  _ORDER_ITEM_QUANT    = "quant"
  _ORDER_ITEM_PRICE    = "price"
  _ORDER_ITEM_FIELDS   = _ORDER_ITEM_ORDER_ID+","+_ORDER_ITEM_ITEM_ID+","+
                         _ORDER_ITEM_QUANT+","+_ORDER_ITEM_PRICE
)

var (
  _ErrGetOrders = errors.New("No se pudo acceder a la lista de órdenes")
  _ErrInsertOrder = errors.New("No se pudo insertar la nueva orden")
  _ErrUpdateOrder = errors.New("No se pudo modificar la orden")
)

func updateOrderField(id int, field string, v any) error {
  _, err := dbExec(
    "UPDATE "+_ORDERS+" "+
    "SET "+field+" = ?, "+ORDER_UPDATED+" = ? "+
    "WHERE "+ORDER_ID+" = ?",
    v, time.Now().Unix(), id,
  )
  if err != nil { return _ErrUpdateOrder }
  return nil
}

func InsertOrder(order Order) error {
  tx, err := _DB.Beginx()
  if err != nil { slog.Error(err.Error()); return _ErrInsertOrder }

  _, err = tx.NamedExec(
    "INSERT INTO "+_ORDERS+" ("+ ORDER_ID+","+_ORDER_USER+","+_ORDER_TOTAL+","+
      ORDER_REF_NUM+","+_ORDER_TIME+","+ORDER_STATUS+") "+
    "VALUES (:"+ORDER_ID+",:"+_ORDER_USER+",:"+_ORDER_TOTAL+",:"+
            ORDER_REF_NUM+",:"+_ORDER_TIME+",:"+ORDER_STATUS+")",
    &order,
  )
  if err != nil { tx.Rollback(); slog.Error(err.Error()); return _ErrInsertOrder}

  for _, line := range order.Lines {
    _, err := tx.Exec(
      "INSERT INTO "+_ORDER_ITEMS+" ("+ _ORDER_ITEM_FIELDS+") "+
      "VALUES (?, ?, ?, ?)",
      order.Id, line.ItemID, line.Quant, line.Price,
    )
    if err != nil {tx.Rollback(); slog.Error(err.Error());return _ErrInsertOrder}
  }

  err = tx.Commit()
  if err != nil { tx.Rollback(); slog.Error(err.Error()); return _ErrInsertOrder}

  return nil
}

type PendingQueueStats struct {
  Count      int   `db:"count"`
  Unreviewed int   `db:"unreviewed"`
  MaxID      int   `db:"max_id"`
  MaxUpdated int64 `db:"max_updated"`
}

func GetPendingQueueStats() (PendingQueueStats, error) {
  var stats PendingQueueStats
  err := dbGet(&stats,
    "SELECT COUNT(*) AS count, COALESCE(SUM("+ORDER_STATUS+" = ?), 0) AS unreviewed, "+
    "COALESCE(MAX("+ORDER_ID+"), 0) AS max_id, "+
    "COALESCE(MAX("+ORDER_UPDATED+"), 0) AS max_updated "+
    "FROM "+_ORDERS+" "+
    "WHERE "+ORDER_STATUS+" IN (?, ?, ?)",
    ORDER_STATUS_UNREVIEWED, ORDER_STATUS_UNREVIEWED, ORDER_STATUS_DENIED, ORDER_STATUS_ACCEPTED,
  )
  if err != nil { return PendingQueueStats{}, errors.New("No se pudo consultar la cola de órdenes") }
  return stats, nil
}

// QueueCounts feeds the staff queue tiles in one query.
type QueueCounts struct {
  Unreviewed     int `db:"unreviewed"`
  Denied         int `db:"denied"`
  Accepted       int `db:"accepted"`
  FulfilledToday int `db:"fulfilled_today"`
}

func GetQueueCounts(since int64) (QueueCounts, error) {
  var c QueueCounts
  err := dbGet(&c,
    "SELECT COALESCE(SUM("+ORDER_STATUS+" = ?), 0) AS unreviewed, "+
    "COALESCE(SUM("+ORDER_STATUS+" = ?), 0) AS denied, "+
    "COALESCE(SUM("+ORDER_STATUS+" = ?), 0) AS accepted, "+
    "COALESCE(SUM("+ORDER_STATUS+" = ? AND "+_ORDER_TIME+" >= ?), 0) AS fulfilled_today "+
    "FROM "+_ORDERS+" "+
    "WHERE "+ORDER_STATUS+" IN (?, ?, ?, ?)",
    ORDER_STATUS_UNREVIEWED, ORDER_STATUS_DENIED, ORDER_STATUS_ACCEPTED,
    ORDER_STATUS_FULFILLED, since,
    ORDER_STATUS_UNREVIEWED, ORDER_STATUS_DENIED, ORDER_STATUS_ACCEPTED, ORDER_STATUS_FULFILLED,
  )
  if err != nil { return QueueCounts{}, errors.New("No se pudo contar las órdenes") }
  return c, nil
}

// DayStats: order count and sales since a time (sales = accepted + fulfilled).
type DayStats struct {
  Orders int     `db:"orders"`
  Sales  float32 `db:"sales"`
}

func GetDayStats(since int64) (DayStats, error) {
  var d DayStats
  err := dbGet(&d,
    "SELECT COUNT(*) AS orders, "+
    "COALESCE(SUM(CASE WHEN "+ORDER_STATUS+" IN (?, ?) THEN "+_ORDER_TOTAL+" ELSE 0 END), 0) AS sales "+
    "FROM "+_ORDERS+" WHERE "+_ORDER_TIME+" >= ?",
    ORDER_STATUS_ACCEPTED, ORDER_STATUS_FULFILLED, since,
  )
  if err != nil { return DayStats{}, errors.New("No se pudo calcular las estadísticas") }
  return d, nil
}

var _OrderSortFields = _SortFields{
  ORDER_ID, _ORDER_USER, _ORDER_TOTAL, _ORDER_TIME, ORDER_STATUS, ORDER_UPDATED,
}

func GetPendingOrders(params *SelectParams) ([]Order, error) {
  orders := []Order{}
  params.Fix(_OrderSortFields)
  err := dbSelect(&orders,
    "SELECT * FROM "+_ORDERS+" "+
    "WHERE "+ORDER_STATUS+" IN (?, ?, ?) "+
    "ORDER BY "+params.Sort+" "+string(params.Dir)+
    " LIMIT ?, ?",
    ORDER_STATUS_UNREVIEWED, ORDER_STATUS_DENIED, ORDER_STATUS_ACCEPTED,
    params.Start, params.Limit,
  )
  if err != nil { return []Order{}, _ErrGetOrders }
  return orders, nil
}

// GetOrderHistory lists closed orders; paged applies params' start and limit.
func GetOrderHistory(params *SelectParams, paged bool) ([]Order, error) {
  orders := []Order{}
  params.Fix(_OrderSortFields)
  query := "SELECT * FROM "+_ORDERS+" "+
    "WHERE "+ORDER_STATUS+" IN (?, ?) "+
    "ORDER BY "+params.Sort+" "+string(params.Dir)
  args := []any{ORDER_STATUS_CANCELLED, ORDER_STATUS_FULFILLED}
  if paged {
    query += " LIMIT ?, ?"
    args = append(args, params.Start, params.Limit)
  }
  err := dbSelect(&orders, query, args...)
  if err != nil { return []Order{}, _ErrGetOrders }
  return orders, nil
}

// OrderItemDetail is one line of an order's breakdown. Price is the unit price
// stored with the order (the item's current price for rows that predate it).
type OrderItemDetail struct {
  Name     string  `db:"name"`
  Price    float32 `db:"price"`
  Quant    int     `db:"quant"`
  Subtotal float32 `db:"subtotal"`
}

func GetOrderItemDetails(orderId int) ([]OrderItemDetail, error) {
  details := []OrderItemDetail{}
  err := _DB.Select(&details,
    "SELECT i."+_ITEM_NAME+" AS name, "+
      "COALESCE(oi."+_ORDER_ITEM_PRICE+", i."+_ITEM_PRICE+") AS price, "+
      "oi."+_ORDER_ITEM_QUANT+" AS quant, "+
      "COALESCE(oi."+_ORDER_ITEM_PRICE+", i."+_ITEM_PRICE+") * oi."+_ORDER_ITEM_QUANT+" AS subtotal "+
    "FROM "+_ORDER_ITEMS+" oi "+
    "JOIN "+_ITEMS+" i ON i."+ITEM_ID+" = oi."+_ORDER_ITEM_ITEM_ID+" "+
    "WHERE oi."+_ORDER_ITEM_ORDER_ID+" = ?",
    orderId,
  )
  if err != nil {
    slog.Error(err.Error())
    return nil, errors.New("No se pudo acceder al desglose de la orden")
  }
  return details, nil
}

func GetOrderFromID(id int) (Order, error) {
  order := Order{}
  err := dbGet(&order,
    "SELECT * "+
    "FROM "+_ORDERS+" "+
    "WHERE "+ORDER_ID+" = ?",
    id,
  )
  if err != nil{
    return Order{},errors.New("No se pudo acceder la orden consultada")
  }

  return order, nil
}

// GetOrdersFromUsername pages through a customer's orders. hasNext comes
// from fetching one row past the page.
func GetOrdersFromUsername(username string, params *SelectParams) (orders []Order, hasNext bool, err error) {
  orders = []Order{}
  params.Fix(_OrderSortFields)
  err = dbSelect(&orders,
    "SELECT * "+
    "FROM "+_ORDERS+" "+
    "WHERE "+_ORDER_USER+" = ? "+
    "ORDER BY "+params.Sort+" "+string(params.Dir)+
    " LIMIT ?, ?",
    username, params.Start, params.Limit+1,
  )
  if err != nil {
    return []Order{}, false, errors.New("No se pudo acceder a la lista de órdenes")
  }
  if len(orders) > params.Limit { orders, hasNext = orders[:params.Limit], true }
  return orders, hasNext, nil
}

func GetLatestOrderFromUsername(username string) (Order, error) {
  order := Order{}
  err := dbGet(&order,
    "SELECT * "+
    "FROM "+_ORDERS+" "+
    "WHERE "+_ORDER_USER+" = ? "+
    "ORDER BY "+_ORDER_TIME+" DESC "+
    "LIMIT 1",
    username,
  )
  if err != nil {
    return Order{},errors.New("No se pudo acceder a la última orden")
  }
  return order, nil
}

func GetLatestOrderID() (int, error) {
  var id int
  err := dbGet(&id,
    "SELECT "+ORDER_ID+" "+
    "FROM "+_ORDERS+" "+
    "ORDER BY "+ORDER_ID+" DESC "+
    "LIMIT 1",
  )
  if err != nil {
    return -1, errors.New("No se pudo acceder al ID de la última orden")
  }
  return id, nil
}


// One UPDATE for a ref edit plus its status change.
func UpdateOrderRefNumAndStatus(id int, refNum string, status OrderStatus) error {
  _, err := dbExec(
    "UPDATE "+_ORDERS+" "+
    "SET "+ORDER_REF_NUM+" = ?, "+ORDER_STATUS+" = ?, "+ORDER_UPDATED+" = ? "+
    "WHERE "+ORDER_ID+" = ?",
    refNum, status, time.Now().Unix(), id,
  )
  if err != nil { return _ErrUpdateOrder }
  return nil
}

func UpdateOrderStatus(id int, status OrderStatus) error {
  return updateOrderField(id, ORDER_STATUS, status)
}

func CancelOrder(id int) error {
  return UpdateOrderStatus(id, ORDER_STATUS_CANCELLED)
}
