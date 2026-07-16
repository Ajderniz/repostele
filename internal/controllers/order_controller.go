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

var _OrderId = 0

func initOrderId() error {
	var err error
	_OrderId, err = models.GetLatestOrderId()
	if err != nil { return err }
	if _OrderId == 0 {
		now, _ := strconv.Atoi(time.Now().Format(_ORDER_DATE_FORMAT))
		_OrderId = now * _ORDER_COUNTER_MAX
	}
	_OrderId++
	return nil
}

func updateOrderId() {
	orderDiv     := float64(_OrderId) / _ORDER_COUNTER_MAX
	orderDate    := int(orderDiv)
	orderCounter := int(math.Round((orderDiv - float64(orderDate)) * _ORDER_COUNTER_MAX))
	now, _ := strconv.Atoi(time.Now().Format(_ORDER_DATE_FORMAT))
	if orderDate < now {
		orderDate    = now
		orderCounter = 1
	} else {
		orderCounter++
	}
	_OrderId = (orderDate * _ORDER_COUNTER_MAX) + orderCounter
}

type _OrderRequest struct {
  RefNum string             `db:"ref_num" json:"ref_num" validate:"required,len=25,numeric"`
  Items  models.ItemIdQuant              `json:"items"   validate:"min=1,max=16,dive,gte=0,lte=4"`
}

var _NoItemsErr = errors.New("Not enough items ordered")

func PostOrder(w http.ResponseWriter, r *http.Request) {
	request := _OrderRequest{}
	if err := bind.JSON(r, &request); err != nil {
		write.ErrorJSON(w, http.StatusBadRequest, err)
		return
	}

	total := 0
	items := models.ItemIdQuant{}
	for itemId, quant := range request.Items {
		item, err := models.GetItemFromId(strconv.Itoa(itemId))
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

	if _OrderId <= 0 { 
		if err := initOrderId(); err != nil {
			write.ErrorJSON(w, http.StatusInternalServerError, err)
			return
		}
	} else if 999 < _OrderId {
		write.ErrorJSON(w, http.StatusInternalServerError, errors.New("Order ID max exceeded"))
		return
	}
	err := models.InsertOrder(models.Order{
		Id:     _OrderId,
		User:   r.Context().Value(models.USER_USERNAME).(string),
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
			models.ORDER_ID: _OrderId,
		},
	})

	updateOrderId()
}

var _CheckOrderNoResults = write.H{"data": "No results found"}

func CheckOrder(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, models.ORDER_ID)
	id, err := strconv.Atoi(idStr);
	if err != nil {
		errman.PrintError(err)
		write.ErrorJSON(w, http.StatusBadRequest, errors.New("Invalid order ID"))
	}

	order, err := models.GetOrderFromId(id)
	if err != nil {
		write.ErrorJSON(w, http.StatusInternalServerError, err)
		return
	}
	if order.User == "" {
		write.JSON(w, http.StatusOK, _CheckOrderNoResults)
		return
	}

	username := r.Context().Value(models.USER_USERNAME).(string)
	if username != order.User {
		write.JSON(w, http.StatusOK, _CheckOrderNoResults)
		return
	}

	write.JSON(w, http.StatusOK, write.H{"data": order})
}