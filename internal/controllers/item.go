package controllers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
	"github.com/ajderniz/repostele/pkg/write"
)

const _ITEM_ID = "id"

func PostItem(w http.ResponseWriter, r *http.Request) {
  post := struct{
    Name      string  `json:"name"      validate:"required"`
    Price     float32 `json:"price"     validate:"required,gte=0"`
    Desc      string  `json:"desc"      validate:"-"`
    ImgPath   string  `json:"img_path"  validate:"-"`
  }{}
  err := bind.JSON(r, &post)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  item := models.Item{
    Name:      post.Name,
    Price:     post.Price,
    TimeMod:   time.Now().Unix(),
    Available: true,
    Desc:      post.Desc,
    ImgPath:   post.ImgPath,
  }
  err = models.InsertItem(item)
  if err != nil { write.Error(w, http.StatusInternalServerError, err); return }

  write.Msg(w, "Item posted successfully")
}

type _MenuData struct {
  Items []models.Item
  Msg   string
}

func GetItems(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  var data _MenuData

  data.Items, err = models.GetItems(params)
  if err != nil { write.Error(w, http.StatusInternalServerError, err); return }
  if len(data.Items) <= 0 { data.Msg = _DataNoResults }

  ctx := context.WithValue(r.Context(), _MAIN_DATA, data)
  ServeTemplateStaff(w, r.WithContext(ctx))
}

func GetItemFromID(w http.ResponseWriter, r *http.Request) {
  idStr := chi.URLParam(r, models.ITEM_ID)
  id, err := strconv.Atoi(idStr)
  if err != nil { 
    write.Error(w, http.StatusBadRequest, _ErrBadSearch)
    return
  }

  item, err := models.GetItemFromID(id)
  if err != nil {
    write.Error(w, http.StatusInternalServerError, err)
    return
  }

  if item.Name == "" {
    write.Data(w, _DataNoResults)
    return
  }

  write.Data(w, item)
}

func UpdateItem(w http.ResponseWriter, r *http.Request) {
  idStr, err := bind.FormValue(r, _ITEM_ID, "required,number")
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }
  id, _ := strconv.Atoi(idStr)

  update := models.ItemUpdate{}
  err = bind.JSON(r, &update)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  err = models.UpdateItem(id, update)
  if err != nil { write.Error(w, http.StatusInternalServerError, err); return }

  write.Msg(w, "Item updated successfully")
}
