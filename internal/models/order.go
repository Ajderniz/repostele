package models

import (
  "errors"
  "time"

  "github.com/ajderniz/repostele/pkg/errman"
)

type OrderStatus int
const (
  ORDER_STATUS_UNREVIEWED OrderStatus = iota
  ORDER_STATUS_DENIED
  ORDER_STATUS_CANCELLED
  ORDER_STATUS_ACCEPTED
  ORDER_STATUS_FULFILLED
)

type ItemIdQuant map[int]int

type Order struct {
  Id      int         `db:"id"       json:"id"`
  User    string      `db:"user"     json:"user"`
  Total   float32     `db:"total"    json:"total"`
  RefNum  string      `db:"ref_num"  json:"ref_num"`
  Time    int64       `db:"time"     json:"time"`
  Status  OrderStatus `db:"status"   json:"status"`
  Updated int64       `db:"updated"  json:"updated"`
  Items  ItemIdQuant                `json:"items"`
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
  //_ORDER_FIELDS  = "user, total, ref_num, time, status"

  _ORDER_ITEMS         = "order_items"
  //_ORDER_ITEM_PK      = "id"
  _ORDER_ITEM_ORDER_ID = "order_id"
  _ORDER_ITEM_ITEM_ID  = "item_id"
  _ORDER_ITEM_QUANT    = "quant"
  _ORDER_ITEM_FIELDS   = _ORDER_ITEM_ORDER_ID+","+_ORDER_ITEM_ITEM_ID+","+
                         _ORDER_ITEM_QUANT
)

var (
  _ErrGetOrders = errors.New("Could not retrieve order list")
  _ErrInsertOrder = errors.New("Could not insert order")
  _ErrUpdateOrder = errors.New("Could not modify the order")
)

func updateOrderField(id int, field string, v any) error {
  _, err := dbBeginExecAndCommit(
    "UPDATE "+_ORDERS+
    "SET ? = ?, "+ORDER_UPDATED+" = ? "+
    "WHERE "+ORDER_ID+" = ?",
    field, v, time.Now().Unix(), id,
  )
  if err != nil { return _ErrUpdateOrder }
  return nil
}

func InsertOrder(order Order) error {
  tx, err := _DB.Beginx()
  if err != nil { errman.PrintError(err); return _ErrInsertOrder }

  _, err = tx.NamedExec(
    "INSERT INTO "+_ORDERS+" "+
    "VALUES (:"+ORDER_ID+",:"+_ORDER_USER+",:"+_ORDER_TOTAL+",:"+
            ORDER_REF_NUM+",:"+_ORDER_TIME+",:"+ORDER_STATUS+")",
    &order,
  )
  if err != nil { errman.PrintError(err); return _ErrInsertOrder }

  for itemId, quant := range order.Items {
    _, err := tx.Exec(
      "INSERT INTO "+_ORDER_ITEMS+" ("+ _ORDER_ITEM_FIELDS+") "+
      "VALUES (?, ?, ?)",
      order.Id, itemId, quant,
    )
    if err != nil { errman.PrintError(err); return _ErrInsertOrder }
  }

  err = tx.Commit()
  if err != nil { errman.PrintError(err); return _ErrInsertOrder }

  return nil
}

var _OrderSortFields = _SortFields{ 
  ORDER_ID, _ORDER_USER, _ORDER_TOTAL, _ORDER_TIME, ORDER_STATUS, ORDER_UPDATED,
}

func GetOrders(params SelectParams) ([]Order, error) {
  orders := []Order{}
  err := dbSelectList(&orders, "*", _ORDERS, params, _OrderSortFields)
  if err != nil { return []Order{}, _ErrGetOrders }
  return orders, nil
}

func getItemsFromOrderID(id int) (ItemIdQuant, error) {
  ois := []struct{
    ItemId int `db:"item_id"`
    Quant  int `db:"quant"`
  }{}
  err := _DB.Select(&ois,
    "SELECT "+_ORDER_ITEM_ITEM_ID+", "+_ORDER_ITEM_QUANT+" "+
    "FROM "+_ORDER_ITEMS+" "+
    "WHERE "+_ORDER_ITEM_ORDER_ID+" = ?",
    id,
  )
  if err != nil {
    errman.PrintError(err)
    return nil, errors.New("Could not retrieve item list from order")
  }

  items := make(ItemIdQuant, len(ois))
  for _, oi := range ois { items[oi.ItemId] = oi.Quant }

  return items, nil
}

func GetOrderFromID(id int) (Order, error) {
  order := Order{}
  err := dbGet(&order,
    "SELECT * "+
    "FROM "+_ORDERS+" "+
    "WHERE "+ORDER_ID+" = ?",
    id,
  )
  if err != nil{return Order{},errors.New("Could not retrieve requested order")}

  order.Items, err = getItemsFromOrderID(id)
  if err != nil { return Order{}, err }

  return order, nil
}

func GetAllOrdersFromUsername(username string) ([]Order, error) {
  orders := []Order{}
  err := dbSelect(&orders,
    "SELECT * "+
    "FROM "+_ORDERS+" "+
    "WHERE "+_ORDER_USER+" = ?"+
    "ORDER BY "+_ORDER_TIME+" DESC",
    username,
  )
  if err != nil { return []Order{},errors.New("Could not retrieve order list") }
  return orders, nil
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
  if err != nil { return Order{},errors.New("Could not retrieve latest order") }
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
  if err != nil { return -1, errors.New("Could not retrieve latest order ID") }
  return id, nil
}


func UpdateOrderRefNum(id int, refNum string) error {
  return updateOrderField(id, ORDER_REF_NUM, refNum)
}

func UpdateOrderStatus(id int, status OrderStatus) error {
  return updateOrderField(id, ORDER_STATUS, status)
}

func CancelOrder(id int) error {
  return UpdateOrderStatus(id, ORDER_STATUS_CANCELLED)
}
