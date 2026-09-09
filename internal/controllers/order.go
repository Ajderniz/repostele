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
)

const (
  _ORDER_DATE_FORMAT = "060102"
  _ORDER_COUNTER_MAX = 1000
)

var (
  _OrderDate    = 0
  _OrderCounter = 0
)

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

var _ErrNoItems = errors.New("No se ordenaron suficientes ítemes")
var _ErrOrderIDMax = errors.New("Se excedió el límite de órdenes diarias")

// PostOrder is reached from the cart page via a plain fetch() call (not
// htmx), which sets HX-Request itself to get the small div-response
// fragment back instead of a full-page render — there's no main-order
// template, so the non-HX path here is (like the rest of /order's
// full-page GETs) only exercised by direct, non-JS requests.
func serveOrderErr(w http.ResponseWriter, r *http.Request, status int, err error) {
  if r.Header.Get("HX-Request") == "true" {
    serveResponseHX(w, err.Error(), status, nil)
    return
  }
  serveErr(w, r, status, err)
}

func serveOrderInternalErr(w http.ResponseWriter, r *http.Request) {
  if r.Header.Get("HX-Request") == "true" {
    serveInternalErrHX(w)
    return
  }
  serveInternalErr(w, r)
}

func PostOrder(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil { serveOrderInternalErr(w, r); return }
  if latestOrder.RefNum != "" &&
     latestOrder.Status != models.ORDER_STATUS_CANCELLED &&
     latestOrder.Status != models.ORDER_STATUS_FULFILLED {
    serveOrderErr(w, r, TooManyRequests, errors.New(
      "Se permite solo una orden pendiente por usuario"),
    )
    return
  }

  request := _OrderRequest{}
  if err := bind.JSON(r, &request); err != nil {
    serveOrderErr(w, r, BadRequest, err)
    return
  }

  var total float32
  items := models.ItemIdQuant{}
  for itemId, quant := range request.Items {
    item, err := models.GetItemFromID(itemId)
    if err != nil  { slog.Error(err.Error()); continue }
    if item.Name == "" { slog.Error("Ítem inválido"); continue }
    total += item.Price * float32(quant)
    items[item.Id] = quant
  }
  if len(items) == 0 {
    slog.Error(_ErrNoItems.Error())
    serveOrderErr(w, r, BadRequest, _ErrNoItems)
    return
  }

  if _OrderDate == 0 || _OrderCounter == 0 {
    if err := initOrderId(); err != nil { serveOrderInternalErr(w, r); return }
  }

  if _ORDER_COUNTER_MAX - 1 < _OrderCounter {
    slog.Error(_ErrOrderIDMax.Error())
    serveOrderInternalErr(w, r)
    return
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
  if err != nil { serveOrderInternalErr(w, r); return }

  updateOrderID()

  if r.Header.Get("HX-Request") == "true" {
    serveResponseHX(w, "Orden enviada. Esperando aprobación.", Created, nil)
    return
  }
  serveResponse(w, r, &_MainData{
    Msg: "Orden enviada. Esperando aprobación.",
    Data: orderId,
    }, Created, nil,
  )
}

func GetAllOrders(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { serveBadRequest(w, r, err); return }

  orders, err := models.GetOrders(&params)
  if err != nil { serveInternalErr(w, r); return }

  _, role, _ := checkSessionUser(r)
  isStaff := role == models.SESSION_ROLE_STAFF

  serveDataHX(
    w,
    r,
    map[string]any{
      "Orders":  orders,
      "IsStaff": isStaff,
      "Params":  params,
      "HasNext": len(orders) == params.Limit,
    },
    "list-orders",
  )
}

func getOrderFromIdUrlParam(r *http.Request) (models.Order, int, error) {
  idStr := chi.URLParam(r, models.ORDER_ID)
  id, err := strconv.Atoi(idStr)
  if err != nil { return models.Order{}, BadRequest, _ErrBadSearch }
  order, err := models.GetOrderFromID(id)
  if err != nil { return models.Order{}, InternalServerError, err }
  return order, OK, nil
}

func serveOrderDetail(w http.ResponseWriter, r *http.Request, order models.Order, isStaff bool) {
  details, err := models.GetOrderItemDetails(order.Id)
  if err != nil { serveOrderInternalErr(w, r); return }
  serveDataHX(w, r, map[string]any{
    "Order":       order,
    "IsStaff":     isStaff,
    "ItemDetails": details,
  }, "table-order")
}

func GetOrderFromID(w http.ResponseWriter, r *http.Request) {
  order, status, err := getOrderFromIdUrlParam(r)
  if err != nil { serveOrderErr(w, r, status, err); return }
  if order.RefNum == "" { serveNoResults(w, r); return }
  serveOrderDetail(w, r, order, true)
}

func GetUserOrderList(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)
  orders, err := models.GetAllOrdersFromUsername(username)
  if err != nil { serveOrderInternalErr(w, r); return }
  serveDataHX(w, r, map[string]any{"Orders": orders, "IsStaff": false}, "list-orders")
}

func CheckUserOrderFromID(w http.ResponseWriter, r *http.Request) {
  idStr := chi.URLParam(r, models.ORDER_ID)
  id, err := strconv.Atoi(idStr)
  if err != nil {
    slog.Error(err.Error())
    serveOrderErr(w, r, BadRequest, errors.New("ID de orden inválido"))
    return
  }

  order, err := models.GetOrderFromID(id)
  if err != nil { serveOrderInternalErr(w, r); return }
  if order.User == "" { serveNoResults(w, r); return }

  username := r.Context().Value(models.USER_USERNAME).(string)
  if username != order.User { serveNoResults(w, r); return }

  serveOrderDetail(w, r, order, false)
}

var _ErrCantModOrder = errors.New("No se puede modificar esta orden")

func GetOrderRefNumEditForm(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil { serveOrderInternalErr(w, r); return }
  if latestOrder.RefNum == "" ||
     (latestOrder.Status != models.ORDER_STATUS_UNREVIEWED &&
      latestOrder.Status != models.ORDER_STATUS_DENIED) {
    serveOrderErr(w, r, Forbidden, _ErrCantModOrder)
    return
  }
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  if err := _Tpl.ExecuteTemplate(w, "form-edit-order-ref", latestOrder); err != nil {
    slog.Error(err.Error())
    serveInternalErrHX(w)
  }
}

func UpdateUserOrderRefNum(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil { serveOrderInternalErr(w, r); return }
  if latestOrder.RefNum == "" ||
     (latestOrder.Status != models.ORDER_STATUS_UNREVIEWED &&
      latestOrder.Status != models.ORDER_STATUS_DENIED) {
    serveOrderErr(w, r, Forbidden, _ErrCantModOrder)
    return
  }
  refNum, err := bind.FormValue(r, models.ORDER_REF_NUM, "required,number,len=25")
  if err != nil { serveOrderErr(w, r, BadRequest, err); return }
  err = models.UpdateOrderRefNum(latestOrder.Id, refNum)
  if err != nil { serveOrderInternalErr(w, r); return }
  // editing puts a denied order back in the staff review queue
  err = models.UpdateOrderStatus(latestOrder.Id, models.ORDER_STATUS_UNREVIEWED)
  if err != nil { serveOrderInternalErr(w, r); return }

  if r.Header.Get("HX-Request") == "true" {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    _Tpl.ExecuteTemplate(w, "div-response", _HXData{Msg: "Se actualizó la orden"})
    _Tpl.ExecuteTemplate(w, "order-ref-num", map[string]any{
      "Id": latestOrder.Id, "RefNum": refNum, "OOB": true,
    })
    _Tpl.ExecuteTemplate(w, "order-status", map[string]any{
      "Id": latestOrder.Id, "Status": models.ORDER_STATUS_UNREVIEWED, "OOB": true,
    })
    return
  }
  serveMsg(w, r, "Se actualizó la orden")
}

func CancelUserOrder(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil { serveOrderInternalErr(w, r); return }
  if latestOrder.RefNum == "" ||
     (latestOrder.Status != models.ORDER_STATUS_UNREVIEWED &&
      latestOrder.Status != models.ORDER_STATUS_DENIED &&
      latestOrder.Status != models.ORDER_STATUS_ACCEPTED) {
    serveOrderErr(w, r, Forbidden, _ErrCantModOrder)
    return
  }
  err = models.CancelOrder(latestOrder.Id)
  if err != nil { serveOrderInternalErr(w, r); return }

  if r.Header.Get("HX-Request") == "true" {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    _Tpl.ExecuteTemplate(w, "div-response", _HXData{Msg: "Se canceló la orden"})
    _Tpl.ExecuteTemplate(w, "oob-delete", "order-"+strconv.Itoa(latestOrder.Id))
    return
  }
  serveMsg(w, r, "Se canceló la orden")
}

func UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
  statusStr, err := bind.FormValue(r, models.ORDER_STATUS, 
    "required,number,gte=0,lte=4",
  )
  if err != nil { serveBadRequestHX(w, err.Error()); return }

  status, _ := strconv.Atoi(statusStr)
  setStatus := models.OrderStatus(status)

  order, httpStatus, err := getOrderFromIdUrlParam(r)
  if err != nil { serveResponseHX(w, err.Error(), httpStatus, nil); return }
  if order.RefNum == "" { serveBadRequestHX(w, "La orden no existe"); return }

  switch order.Status {
  case models.ORDER_STATUS_UNREVIEWED, models.ORDER_STATUS_DENIED:
    if setStatus != models.ORDER_STATUS_ACCEPTED &&
       setStatus != models.ORDER_STATUS_DENIED {
      serveBadRequestHX(w, _ErrCantModOrder.Error())
      return
    }
  case models.ORDER_STATUS_ACCEPTED:
    if setStatus != models.ORDER_STATUS_FULFILLED {
      serveBadRequestHX(w, _ErrCantModOrder.Error())
      return
    }
  default: serveBadRequestHX(w, _ErrCantModOrder.Error()); return
  }

  err = models.UpdateOrderStatus(order.Id, setStatus)
  if err != nil { serveInternalErrHX(w); return }

  serveResponseHX(w, "Se actualizó el estado de la orden", OK, nil)
}
