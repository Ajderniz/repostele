package models

import (
	"database/sql"
	"errors"
	"strconv"

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
  Id     int         `db:"id"      json:"id"`
  User   string      `db:"user"    json:"user"`
  Total  int         `db:"total"   json:"total"`
  RefNum string      `db:"ref_num" json:"ref_num"`
  Time   int64       `db:"time"    json:"time"`
  Status OrderStatus `db:"status"  json:"status"`
  Items  ItemIdQuant              `json:"items"`
}

const (
  _ORDERS        = "orders"
  ORDER_ID       = "id"
  _ORDER_USER    = "user"
  _ORDER_TOTAL   = "total"
  ORDER_REF_NUM = "ref_num"
  _ORDER_TIME    = "time"
  _ORDER_STATUS  = "status"
  //_ORDER_FIELDS  = "user, total, ref_num, time, status"

  _ORDER_ITEMS         = "order_items"
  //_ORDER_ITEM_PK      = "id"
  _ORDER_ITEM_ORDER_ID = "order_id"
  _ORDER_ITEM_ITEM_ID  = "item_id"
  _ORDER_ITEM_QUANT    = "quant"
  _ORDER_ITEM_FIELDS   = _ORDER_ITEM_ORDER_ID+","+_ORDER_ITEM_ITEM_ID+","+
                         _ORDER_ITEM_QUANT
)

var _InsertOrderErr = errors.New("Could not insert order")

func InsertOrder(order Order) error {
  tx, err := _DB.Beginx()
  if err != nil { errman.PrintError(err); return _InsertOrderErr }

  _, err = tx.NamedExec(
    "INSERT INTO "+_ORDERS+" "+
    "VALUES (:"+ORDER_ID+",:"+_ORDER_USER+",:"+_ORDER_TOTAL+",:"+
            ORDER_REF_NUM+",:"+_ORDER_TIME+",:"+_ORDER_STATUS+")",
    &order,
  )
  if err != nil { errman.PrintError(err); return _InsertOrderErr }

  for itemId, quant := range order.Items {
    _, err := tx.Exec(
      "INSERT INTO "+_ORDER_ITEMS+" ("+ _ORDER_ITEM_FIELDS+") "+
      "VALUES ($1, $2, $3)",
      order.Id, itemId, quant,
    )
    if err != nil { errman.PrintError(err); return _InsertOrderErr }
  }

  err = tx.Commit()
  if err != nil { errman.PrintError(err); return _InsertOrderErr }

  return nil
}

func getItemsFromOrderID(id int) (ItemIdQuant, error) {
  ois := []struct{
    ItemId int `db:"item_id"`
    Quant  int `db:"quant"`
  }{}
  err := _DB.Select(&ois,
    "SELECT "+_ORDER_ITEM_ITEM_ID+", "+_ORDER_ITEM_QUANT+" "+
    "FROM "+_ORDER_ITEMS+" "+
    "WHERE "+_ORDER_ITEM_ORDER_ID+" = $1",
    id,
  )
  if err != nil {
    errman.PrintError(err)
    return nil, errors.New("Could not retreive item list from order")
  }

  items := make(ItemIdQuant, len(ois))
  for _, oi := range ois { items[oi.ItemId] = oi.Quant }

  return items, nil
}

var _GetOrderFromIdError = errors.New("Could not retreive requested order")

func GetOrderFromID(id int) (Order, error) {
  order := Order{}
  err := _DB.Get(&order,
    "SELECT * "+
    "FROM "+_ORDERS+" "+
    "WHERE "+ORDER_ID+" = $1",
    id,
  )
  if err != nil {
    if err == sql.ErrNoRows { return Order{}, nil }
    errman.PrintError(err)
    return Order{}, _GetOrderFromIdError
  }

  order.Items, err = getItemsFromOrderID(id)
  if err != nil { return Order{}, err }

  return order, nil
}

func GetAllOrdersFromUsername(username string) ([]Order, error) {
  orders := []Order{}
  err := _DB.Select(&orders,
    "SELECT * "+
    "FROM "+_ORDERS+" "+
    "WHERE "+_ORDER_USER+" = $1",
    username,
  )
  if err != nil {
    if err == sql.ErrNoRows { return []Order{}, nil }
    errman.PrintError(err)
    return []Order{}, errors.New("Could not retreive order list")
  }
  return orders, nil
}

func GetLatestOrderFromUsername(username string) (Order, error) {
  order := Order{}
  err := _DB.Get(&order,
    "SELECT "+_ORDER_STATUS+" "+
    "FROM "+_ORDERS+" "+
    "WHERE "+_ORDER_USER+" = $1 "+
    "ORDER BY "+_ORDER_TIME+" DESC "+
    "LIMIT 1",
    username,
  )
  if err != nil {
    if err == sql.ErrNoRows { return Order{}, nil }
    errman.PrintError(err)
    return Order{}, errors.New("Could not retreive latest order status")
  }
  return order, nil
}

func GetLatestOrderID() (int, error) {
  var id int
  err := _DB.Get(&id,
    "SELECT "+ORDER_ID+" "+
    "FROM "+_ORDERS+" "+
    "ORDER BY "+ORDER_ID+" DESC "+
    "LIMIT 1",
  )
  if err != nil {
    if err == sql.ErrNoRows { return 0, nil }
    errman.PrintError(err)
    return -1, errors.New("Could not retreive latest order ID")
  }
  return id, nil
}

var _UpdateOrderRefNumErr = errors.New("Could not modify the order")

func UpdateOrderRefNum(id int, refNum string) error {
  tx, err := _DB.Beginx()
  if err != nil { errman.PrintError(err); return _UpdateOrderRefNumErr }

  _, err = tx.Exec(
    "UPDATE "+_ORDERS+" "+
    "SET "+ORDER_REF_NUM+" = $1 "+
    "WHERE "+ORDER_ID+" = $2",
    refNum, id,
  )
  if err != nil { errman.PrintError(err); return _UpdateOrderRefNumErr }

  err = tx.Commit()
  if err != nil { errman.PrintError(err); return _UpdateOrderRefNumErr }

  return nil
}

var _CancelledStr = strconv.Itoa(int(ORDER_STATUS_CANCELLED))

func CancelOrder(id int) error {
  tx, err := _DB.Beginx()
  if err != nil { errman.PrintError(err); return _UpdateOrderRefNumErr }

  _, err = tx.Exec(
    "UPDATE "+_ORDERS+" "+
    "SET "+_ORDER_STATUS+" = "+_CancelledStr+
    "WHERE +"+ORDER_ID+" = $1",
    id,
  )
  if err != nil { errman.PrintError(err); return _UpdateOrderRefNumErr }
  
  err = tx.Commit()
  if err != nil { errman.PrintError(err); return _UpdateOrderRefNumErr }

  return nil
}