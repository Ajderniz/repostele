package controllers

import (
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
	"github.com/ajderniz/repostele/pkg/write"
)

const (
  _ORDER_DATE_FORMAT = "060102"
  _ORDER_COUNTER_MAX = 1000
)

var _OrderDate    = 0
var _OrderCounter = 0

func orderId2DateAndCounter(orderId int) (date, counter int) {
  div     := float64(orderId) / _ORDER_COUNTER_MAX
  date    = int(div)
  counter = int(math.Round((div - float64(date)) * _ORDER_COUNTER_MAX))
  return
}

func orderDateAndCounter2ID(date, counter int) int {
  return (date * _ORDER_COUNTER_MAX) + counter
}

func getOrderDateInt() (date int) {
  date, _ = strconv.Atoi(time.Now().Format(_ORDER_DATE_FORMAT))
  return
}

func initOrderId() error {
  orderId, err := models.GetLatestOrderID()
  if err != nil { return err }
  if orderId == 0 {
    _OrderDate    = getOrderDateInt()
    _OrderCounter = 1
  } else {
    _OrderDate, _OrderCounter = orderId2DateAndCounter(orderId)
    updateOrderID()
  }
  return nil
}

func updateOrderID() {
  now := getOrderDateInt()
  if _OrderDate < now {
    _OrderDate    = now
    _OrderCounter = 1
  } else {
    _OrderCounter++
  }
}

type _OrderRequest struct {
  RefNum string             `db:"ref_num" json:"ref_num" validate:"required,len=25,numeric"`
  Items  models.ItemIdQuant              `json:"items"   validate:"min=1,max=16,dive,gte=0,lte=4"`
}

var _ErrNoItems = errors.New("Not enough items ordered")
var _ErrOrderIDMax = errors.New("Order ID max exceeded")

func PostOrder(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil {
    write.Error(w, http.StatusInternalServerError, err)
    return
  }
  if latestOrder.Status != models.ORDER_STATUS_CANCELLED &&
     latestOrder.Status != models.ORDER_STATUS_FULFILLED {
    write.Error(w, http.StatusTooManyRequests, errors.New("Only one order per user"))
    return
  }

  request := _OrderRequest{}
  if err := bind.JSON(r, &request); err != nil {
    write.Error(w, http.StatusBadRequest, err)
    return
  }

  var total float32
  items := models.ItemIdQuant{}
  for itemId, quant := range request.Items {
    item, err := models.GetItemFromID(itemId)
    if err != nil  {
      slog.Error(err.Error())
      continue
    }
    if item.Name == "" {
      slog.Error("Invalid item requested")
      continue
    }
    total += item.Price * float32(quant)
    items[item.Id] = quant
  }
  if len(items) == 0 {
    slog.Error(_ErrNoItems.Error())
    write.Error(w, http.StatusBadRequest, _ErrNoItems)
    return
  }

  if _OrderDate == 0 || _OrderCounter == 0 { 
    if err := initOrderId(); err != nil {
      write.Error(w, http.StatusInternalServerError, err)
      return
    }
  }

  if _ORDER_COUNTER_MAX - 1 < _OrderCounter {
    slog.Error(_ErrOrderIDMax.Error())
    write.Error(w, http.StatusInternalServerError, _ErrOrderIDMax)
  }

  orderId := orderDateAndCounter2ID(_OrderDate, _OrderCounter)
  err = models.InsertOrder(models.Order{
    Id:     orderId,
    User:   username,
    Total:  total,
    RefNum: request.RefNum,
    Time:   time.Now().Unix(),
    Status: models.ORDER_STATUS_UNREVIEWED,
    Items:  items,
  })
  if err != nil {
    write.Error(w, http.StatusInternalServerError, err)
    return
  }

  write.JSON(w, http.StatusCreated, write.H{
    write.KEY_MSG: "Order posted. Waiting for approval",
    write.KEY_DAT: write.H{
      models.ORDER_ID: orderId,
    },
  })

  updateOrderID()
}

func GetAllOrders(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  orders, err := models.GetOrders(params)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}
  if len(orders) <= 0 { write.Data(w, _DataNoResults); return }

  write.Data(w, orders)
}

func getOrderFromIdUrlParam(r *http.Request) (models.Order, int, error) {
  idStr := chi.URLParam(r, models.ORDER_ID)
  id, err := strconv.Atoi(idStr)
  if err != nil { return models.Order{}, http.StatusBadRequest, _ErrBadSearch }
  order, err := models.GetOrderFromID(id)
  if err != nil { return models.Order{}, http.StatusInternalServerError, err }
  return order, http.StatusOK, nil
}

func GetOrderFromID(w http.ResponseWriter, r *http.Request) {
  order, status, err := getOrderFromIdUrlParam(r)
  if err != nil { write.Error(w, status, err); return }
  if order.RefNum == "" { write.Data(w, _DataNoResults); return }
  write.Data(w, order)
}

func GetUserOrderList(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)
  orders, err := models.GetAllOrdersFromUsername(username)
  if err != nil {
    write.Error(w, http.StatusInternalServerError, err)
    return
  }
  if len(orders) == 0 { write.Data(w, _DataNoResults); return }
  write.Data(w, orders)
}

func CheckUserOrderFromID(w http.ResponseWriter, r *http.Request) {
  idStr := chi.URLParam(r, models.ORDER_ID)
  id, err := strconv.Atoi(idStr);
  if err != nil {
    slog.Error(err.Error())
    write.Error(w, http.StatusBadRequest, errors.New("Invalid order ID"))
  }

  order, err := models.GetOrderFromID(id)
  if err != nil {
    write.Error(w, http.StatusInternalServerError, err)
    return
  }
  if order.User == "" { write.Data(w, _DataNoResults); return }

  username := r.Context().Value(models.USER_USERNAME).(string)
  if username != order.User { write.Data(w, _DataNoResults); return }

  write.Data(w, order)
}

var _ErrCantModOrder = errors.New("Cannot modify this order")

func UpdateUserOrderRefNum(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil {
    write.Error(w, http.StatusInternalServerError, err)
    return
  }
  if latestOrder.Status != models.ORDER_STATUS_UNREVIEWED &&
     latestOrder.Status != models.ORDER_STATUS_DENIED {
    write.Error(w, http.StatusForbidden, _ErrCantModOrder)
    return
  }
  refNum, err := bind.FormValue(r,models.ORDER_REF_NUM,"required,number,len=25")
  if err != nil {
    write.Error(w, http.StatusBadRequest, err)
    return
  }
  err = models.UpdateOrderRefNum(latestOrder.Id, refNum)
  if err != nil {
    write.Error(w, http.StatusInternalServerError, err)
    return
  }
  write.Msg(w, "Order updated successfully")
}

func CancelUserOrder(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil {
    write.Error(w, http.StatusInternalServerError, err)
    return
  }
  if latestOrder.Status != models.ORDER_STATUS_UNREVIEWED &&
     latestOrder.Status != models.ORDER_STATUS_DENIED && 
     latestOrder.Status != models.ORDER_STATUS_ACCEPTED {
    write.Error(w, http.StatusForbidden, _ErrCantModOrder)
    return
  }
  err = models.CancelOrder(latestOrder.Id)
  if err != nil {
    write.Error(w, http.StatusInternalServerError, err)
    return
  }
  write.Msg(w, "The order was cancelled")
}

func UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
  statusStr, err := bind.FormValue(r, models.ORDER_STATUS, 
    "required,number,gte=0,lte=4",
  )
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }
  status, _ := strconv.Atoi(statusStr)

  setStatus := models.OrderStatus(status)

  order, httpStatus, err := getOrderFromIdUrlParam(r)
  if err != nil { write.Error(w, httpStatus, err); return }
  if order.RefNum == "" { 
    write.Error(w, http.StatusBadRequest,errors.New("Order does not exist"))
    return
  }

  switch order.Status {
  case models.ORDER_STATUS_UNREVIEWED, models.ORDER_STATUS_DENIED:
    if setStatus != models.ORDER_STATUS_ACCEPTED &&
       setStatus != models.ORDER_STATUS_DENIED {
      write.Error(w, http.StatusBadRequest, _ErrCantModOrder)
      return
    }
  case models.ORDER_STATUS_ACCEPTED:
    if setStatus != models.ORDER_STATUS_FULFILLED {
      write.Error(w, http.StatusBadRequest, _ErrCantModOrder)
      return
    }
  default:
    write.Error(w, http.StatusBadRequest, _ErrCantModOrder)
    return
  }

  err = models.UpdateOrderStatus(order.Id, setStatus)
  if err != nil { write.Error(w, http.StatusInternalServerError,err);return}

  write.Msg(w, "Order status updated successfully")
}
