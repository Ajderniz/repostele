package models

import (
  "database/sql"
  "errors"

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
  Items  ItemIdQuant
}

const (
  _ORDERS        = "orders"
  ORDER_ID       = "id"
  _ORDER_USER    = "user"
  _ORDER_TOTAL   = "total"
  _ORDER_REF_NUM = "ref_num"
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
  tx := _DB.MustBegin()

  _, err := tx.NamedExec(
    "INSERT INTO "+_ORDERS+" "+
    "VALUES (:"+ORDER_ID+",:"+_ORDER_USER+",:"+_ORDER_TOTAL+",:"+
            _ORDER_REF_NUM+",:"+_ORDER_TIME+",:"+_ORDER_STATUS+")",
    &order,
  )
  if err != nil {
    errman.PrintError(err)
    return _InsertOrderErr
  }

  for itemId, quant := range order.Items {
    _, err := tx.Exec(
      "INSERT INTO "+_ORDER_ITEMS+" ("+ _ORDER_ITEM_FIELDS+") "+
      "VALUES ($1, $2, $3)",
      order.Id, itemId, quant,
    )
    if err != nil {
      errman.PrintError(err)
      return _InsertOrderErr
    }
  }

  err = tx.Commit()
  if err != nil {
    errman.PrintError(err)
    return _InsertOrderErr
  }
  return nil
}

var _GetOrderFromIdError = errors.New("Could not retreive requested order")

func GetOrderFromId(id int) (Order, error) {
  order := Order{}

  err := _DB.Get(&order,
    "SELECT * " +
    "FROM "+_ORDERS+" "+
    "WHERE "+ORDER_ID+" = $1",
    id,
  )
  if err != nil {
    if err == sql.ErrNoRows { return Order{}, nil }
    errman.PrintError(err)
    return Order{}, _GetOrderFromIdError
  }

  ois := []struct{
    ItemId int `db:"item_id"`
    Quant  int `db:"quant"`
  }{}
  err = _DB.Select(&ois,
    "SELECT "+_ORDER_ITEM_ITEM_ID+", "+_ORDER_ITEM_QUANT+" "+
    "FROM "+_ORDER_ITEMS+" "+
    "WHERE "+_ORDER_ITEM_ORDER_ID+" = $1",
    id,
  )
  if err != nil {
    errman.PrintError(err)
    return Order{}, _GetOrderFromIdError
  }
  order.Items = make(ItemIdQuant, len(ois))
  for _, oi := range ois { order.Items[oi.ItemId] = oi.Quant }

  return order, nil
}

func GetLatestOrderId() (int, error) {
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
    return -1, errors.New("Could not retreive order list")
  }
  return id, nil
}