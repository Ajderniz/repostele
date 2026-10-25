package controllers

import (
	"encoding/csv"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
)

const (
  _ORDER_DATE_FORMAT = "060102"
  _ORDER_COUNTER_MAX = 1000
)

// Order IDs are YYMMDD*1000 + a daily counter. _OrderDate/_OrderCounter hold
// the last ID handed out; the zero date means "not loaded from the DB yet".
var (
  _OrderIDMu    sync.Mutex
  _OrderDate    = 0
  _OrderCounter = 0
)

func getOrderDateInt() (date int) {
  date, _ = strconv.Atoi(time.Now().Format(_ORDER_DATE_FORMAT))
  return
}

// nextOrderID reserves the next ID. A failed insert leaves a gap, never a reuse.
func nextOrderID() (int, error) {
  _OrderIDMu.Lock()
  defer _OrderIDMu.Unlock()

  if _OrderDate == 0 {
    latest, err := models.GetLatestOrderID()
    if err != nil { return 0, err }
    _OrderDate, _OrderCounter = latest/_ORDER_COUNTER_MAX, latest%_ORDER_COUNTER_MAX
  }
  if today := getOrderDateInt(); _OrderDate < today {
    _OrderDate, _OrderCounter = today, 0
  }
  if _ORDER_COUNTER_MAX-1 <= _OrderCounter { return 0, _ErrOrderIDMax }

  _OrderCounter++
  return _OrderDate*_ORDER_COUNTER_MAX + _OrderCounter, nil
}

type _OrderRequest struct {
  RefNum string             `db:"ref_num" json:"ref_num" validate:"required,len=25,numeric"`
  Items  models.ItemIdQuant              `json:"items"   validate:"min=1,max=16,dive,gte=0,lte=4"`
}

var _ErrNoItems = errors.New("No se ordenaron suficientes ítemes")
var _ErrOrderIDMax = errors.New("Se excedió el límite de órdenes diarias")

// PostOrder is reached from the cart page via a plain fetch() call (not
// htmx), which sets HX-Request itself to get the small div-response
// fragment back instead of a full-page render.
func PostOrder(w http.ResponseWriter, r *http.Request) {
  username := ctxUsername(r)
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil { serveInternalErrAuto(w, r); return }
  if latestOrder.Pending() {
    serveErrAuto(w, r, TooManyRequests, errors.New(
      "Se permite solo una orden pendiente por usuario"),
    )
    return
  }

  request := _OrderRequest{}
  if err := bind.JSON(r, &request); err != nil {
    serveErrAuto(w, r, BadRequest, err)
    return
  }

  ids := make([]int, 0, len(request.Items))
  for itemId := range request.Items { ids = append(ids, itemId) }
  found, err := models.GetItemsFromIDs(ids)
  if err != nil { serveInternalErrAuto(w, r); return }

  var total float32
  items := models.ItemIdQuant{}
  for itemId, quant := range request.Items {
    item, ok := found[itemId]
    if !ok || item.Name == "" { slog.Error("Ítem inválido"); continue }
    total += item.Price * float32(quant)
    items[item.Id] = quant
  }
  if len(items) == 0 {
    slog.Error(_ErrNoItems.Error())
    serveErrAuto(w, r, BadRequest, _ErrNoItems)
    return
  }

  orderId, err := nextOrderID()
  if err != nil { slog.Error(err.Error()); serveInternalErrAuto(w, r); return }

  err = models.InsertOrder(models.Order{
    Id:     orderId,
    User:   username,
    Total:  total,
    RefNum: request.RefNum,
    Time:   time.Now().Unix(),
    Status: models.ORDER_STATUS_UNREVIEWED,
    Items:  items,
  })
  if err != nil { serveInternalErrAuto(w, r); return }

  if isHX(r) {
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

  orders, err := models.GetPendingOrders(&params)
  if err != nil { serveInternalErr(w, r); return }

  serveDataHX(
    w,
    r,
    map[string]any{
      "Orders":   orders,
      "IsStaff":  true,
      "Params":   params,
      "HasNext":  len(orders) == params.Limit,
      "Endpoint": "/dashboard/orders",
      "Target":   "#queue-list",
    },
    "list-orders",
  )
}

func GetOrderHistory(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { serveBadRequest(w, r, err); return }

  orders, err := models.GetOrderHistory(&params, true)
  if err != nil { serveInternalErr(w, r); return }

  // A full-panel swap (tab, sort, paging) gets the list plus its aside; an
  // order-changed reload targets #dash-list and gets the list alone.
  tpl := "history-layout"
  if r.Header.Get("HX-Target") == "dash-list" { tpl = "list-orders" }
  serveDataHX(
    w,
    r,
    map[string]any{
      "Orders":      orders,
      "IsStaff":     true,
      "ShowHistory": true,
      "Params":      params,
      "HasNext":     len(orders) == params.Limit,
      "Endpoint":    "/dashboard/orders/history",
      "Target":      "#dash-panel",
    },
    tpl,
  )
}

func ExportOrderHistory(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { serveBadRequest(w, r, err); return }

  orders, err := models.GetOrderHistory(&params, false)
  if err != nil { serveInternalErr(w, r); return }

  w.Header().Set("Content-Type", "text/csv; charset=utf-8")
  w.Header().Set("Content-Disposition", `attachment; filename="historial-ordenes.csv"`)

  cw := csv.NewWriter(w)
  cw.Write([]string{"ID", "Usuario", "Total", "Núm. referencia", "Fecha", "Estado"})
  for _, o := range orders {
    cw.Write([]string{
      strconv.Itoa(o.Id),
      o.User,
      strconv.FormatFloat(float64(o.Total), 'f', 2, 32),
      o.RefNum,
      time.Unix(o.Time, 0).Format("2006-01-02 15:04:05"),
      orderStatusName(o.Status),
    })
  }
  cw.Flush()
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
  if err != nil { serveInternalErrAuto(w, r); return }
  serveDataHX(w, r, map[string]any{
    "Order":       order,
    "IsStaff":     isStaff,
    "ItemDetails": details,
  }, "table-order")
}

func GetOrderFromID(w http.ResponseWriter, r *http.Request) {
  order, status, err := getOrderFromIdUrlParam(r)
  if err != nil { serveErrAuto(w, r, status, err); return }
  if order.RefNum == "" { serveNoResults(w, r); return }
  serveOrderDetail(w, r, order, true)
}

func GetUserOrderList(w http.ResponseWriter, r *http.Request) {
  username := ctxUsername(r)
  orders, err := models.GetAllOrdersFromUsername(username)
  if err != nil { serveInternalErrAuto(w, r); return }
  serveDataHX(w, r, map[string]any{"Orders": orders, "IsStaff": false, "ShowHistory": true}, "list-orders")
}

func CheckUserOrderFromID(w http.ResponseWriter, r *http.Request) {
  idStr := chi.URLParam(r, models.ORDER_ID)
  id, err := strconv.Atoi(idStr)
  if err != nil {
    slog.Error(err.Error())
    serveErrAuto(w, r, BadRequest, errors.New("ID de orden inválido"))
    return
  }

  order, err := models.GetOrderFromID(id)
  if err != nil { serveInternalErrAuto(w, r); return }
  if order.User == "" { serveNoResults(w, r); return }

  username := ctxUsername(r)
  if username != order.User { serveNoResults(w, r); return }

  serveOrderDetail(w, r, order, false)
}

var _ErrCantModOrder = errors.New("No se puede modificar esta orden")

func GetOrderRefNumEditForm(w http.ResponseWriter, r *http.Request) {
  username := ctxUsername(r)
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil { serveInternalErrAuto(w, r); return }
  if !latestOrder.Editable() {
    serveErrAuto(w, r, Forbidden, _ErrCantModOrder)
    return
  }
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  if err := _Tpl.ExecuteTemplate(w, "order-ref-edit", latestOrder); err != nil {
    slog.Error(err.Error())
    serveInternalErrHX(w)
  }
}

func GetOrderRefView(w http.ResponseWriter, r *http.Request) {
  username := ctxUsername(r)
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil { serveInternalErrAuto(w, r); return }
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  if err := _Tpl.ExecuteTemplate(w, "order-ref-slot", latestOrder); err != nil {
    slog.Error(err.Error())
    serveInternalErrHX(w)
  }
}

func UpdateUserOrderRefNum(w http.ResponseWriter, r *http.Request) {
  username := ctxUsername(r)
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil { serveInternalErrAuto(w, r); return }
  if !latestOrder.Editable() {
    serveErrAuto(w, r, Forbidden, _ErrCantModOrder)
    return
  }
  refNum, err := bind.FormValue(r, models.ORDER_REF_NUM, "required,number,len=25")
  if err != nil { serveErrAuto(w, r, BadRequest, err); return }
  // editing puts a denied order back in the staff review queue
  err = models.UpdateOrderRefNumAndStatus(latestOrder.Id, refNum, models.ORDER_STATUS_UNREVIEWED)
  if err != nil { serveInternalErrAuto(w, r); return }

  if isHX(r) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    _Tpl.ExecuteTemplate(w, "order-ref-slot", map[string]any{
      "Id": latestOrder.Id, "RefNum": refNum,
    })
    _Tpl.ExecuteTemplate(w, "toast", "Se actualizó la orden")
    _Tpl.ExecuteTemplate(w, "order-status", map[string]any{
      "Id": latestOrder.Id, "Status": models.ORDER_STATUS_UNREVIEWED, "OOB": true,
    })
    return
  }
  serveMsg(w, r, "Se actualizó la orden")
}

func CancelUserOrder(w http.ResponseWriter, r *http.Request) {
  username := ctxUsername(r)
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil { serveInternalErrAuto(w, r); return }
  if !latestOrder.Pending() {
    serveErrAuto(w, r, Forbidden, _ErrCantModOrder)
    return
  }
  err = models.CancelOrder(latestOrder.Id)
  if err != nil { serveInternalErrAuto(w, r); return }

  if isHX(r) {
    // Row stays (customer list shows history); status and actions update in place.
    serveOOB(w, OK, _HXData{Msg: "Se canceló la orden"},
      frag("order-status", map[string]any{"Id": latestOrder.Id, "Status": models.ORDER_STATUS_CANCELLED, "OOB": true}),
      frag("order-actions", map[string]any{
        "Id": latestOrder.Id, "Status": models.ORDER_STATUS_CANCELLED, "IsStaff": false, "OOB": true,
      }),
    )
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

  // Rows update in place; the order lists re-sync from the server on order-changed.
  serveOOB(w, OK, _HXData{Msg: "Se actualizó el estado de la orden"},
    frag("order-status", map[string]any{"Id": order.Id, "Status": setStatus, "OOB": true}),
    frag("order-actions", map[string]any{
      "Id": order.Id, "Status": setStatus, "IsStaff": true, "OOB": true,
    }),
  )
}
