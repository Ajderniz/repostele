package controllers

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
	"github.com/ajderniz/repostele/pkg/errman"
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

var _NoItemsErr = errors.New("Not enough items ordered")
var _OrderIDMaxErr = errors.New("Order ID max exceeded")

func PostOrder(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil {
    write.ErrorJSON(w, http.StatusInternalServerError, err)
    return
  }
  if latestOrder.Status != models.ORDER_STATUS_CANCELLED &&
     latestOrder.Status != models.ORDER_STATUS_FULFILLED {
    write.ErrorJSON(w, http.StatusTooManyRequests, errors.New("Only one order per user"))
    return
  }

  request := _OrderRequest{}
  if err := bind.JSON(r, &request); err != nil {
    write.ErrorJSON(w, http.StatusBadRequest, err)
    return
  }

  total := 0
  items := models.ItemIdQuant{}
  for itemId, quant := range request.Items {
    item, err := models.GetItemFromID(strconv.Itoa(itemId))
    if err != nil  {
      errman.PrintError(err)
      continue
    }
    if item.Name == "" {
      errman.PrintError(errors.New("Invalid item requested"))
      continue
    }
    total += item.Price * quant
    items[item.Id] = quant
  }
  if len(items) == 0 {
    errman.PrintError(_NoItemsErr)
    write.ErrorJSON(w, http.StatusBadRequest, _NoItemsErr)
    return
  }

  if _OrderDate == 0 || _OrderCounter == 0 { 
    if err := initOrderId(); err != nil {
      write.ErrorJSON(w, http.StatusInternalServerError, err)
      return
    }
  }

  if _ORDER_COUNTER_MAX - 1 < _OrderCounter {
  	errman.PrintError(_OrderIDMaxErr)
    write.ErrorJSON(w, http.StatusInternalServerError, _OrderIDMaxErr)
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
    write.ErrorJSON(w, http.StatusInternalServerError, err)
    return
  }

  write.JSON(w, http.StatusCreated, write.H{
    "message": "Order posted. Waiting for approval",
    "data": write.H{
      models.ORDER_ID: orderId,
    },
  })

  updateOrderID()
}

var _DataNoResults = write.H{"data": "No results found"}

func GetOrderList(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)
  orders, err := models.GetAllOrdersFromUsername(username)
  if err != nil {
    write.ErrorJSON(w, http.StatusInternalServerError, err)
    return
  }
  if len(orders) == 0 {
    write.JSON(w, http.StatusOK, _DataNoResults)
    return
  }
  write.JSON(w, http.StatusOK, write.H{"data": orders})
}

func CheckOrderFromID(w http.ResponseWriter, r *http.Request) {
  idStr := chi.URLParam(r, models.ORDER_ID)
  id, err := strconv.Atoi(idStr);
  if err != nil {
    errman.PrintError(err)
    write.ErrorJSON(w, http.StatusBadRequest, errors.New("Invalid order ID"))
  }

  order, err := models.GetOrderFromID(id)
  if err != nil {
    write.ErrorJSON(w, http.StatusInternalServerError, err)
    return
  }
  if order.User == "" {
    write.JSON(w, http.StatusOK, _DataNoResults)
    return
  }

  username := r.Context().Value(models.USER_USERNAME).(string)
  if username != order.User {
    write.JSON(w, http.StatusOK, _DataNoResults)
    return
  }

  write.JSON(w, http.StatusOK, write.H{"data": order})
}

var _CannotModifyOrderErr = errors.New("Cannot modify this order")

func UpdateOrderRefNum(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil {
    write.ErrorJSON(w, http.StatusInternalServerError, err)
    return
  }
  if latestOrder.Status != models.ORDER_STATUS_UNREVIEWED &&
     latestOrder.Status != models.ORDER_STATUS_DENIED {
    write.ErrorJSON(w, http.StatusForbidden, _CannotModifyOrderErr)
    return
  }
  refNum := ""
  err = bind.FormValue(r, &refNum, models.ORDER_REF_NUM, "required,len=25")
  if err != nil {
  	write.ErrorJSON(w, http.StatusBadRequest, err)
  	return
  }
  err = models.UpdateOrderRefNum(latestOrder.Id, refNum)
  if err != nil {
  	write.ErrorJSON(w, http.StatusInternalServerError, err)
  	return
  }
  write.JSON(w, http.StatusOK, write.H{"message": "Order updated successfully"})
}

func CancelOrder(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil {
    write.ErrorJSON(w, http.StatusInternalServerError, err)
    return
  }
  if latestOrder.Status != models.ORDER_STATUS_UNREVIEWED &&
     latestOrder.Status != models.ORDER_STATUS_DENIED && 
     latestOrder.Status != models.ORDER_STATUS_ACCEPTED {
    write.ErrorJSON(w, http.StatusForbidden, _CannotModifyOrderErr)
    return
  }
  err = models.CancelOrder(latestOrder.Id)
  if err != nil {
  	write.ErrorJSON(w, http.StatusInternalServerError, err)
  	return
  }
  write.JSON(w, http.StatusOK, write.H{"message": "The order was cancelled"})
}