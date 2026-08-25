package controllers

import (
	"net/http"
  "log/slog"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
  "github.com/ajderniz/repostele/pkg/upload"
  "github.com/ajderniz/repostele/static"
)

const (
  _MAX_UPLOAD_MEM = 10 << 20 // 10 MiB, incl. non-file form fields
)

func PostItem(w http.ResponseWriter, r *http.Request) {
  err := r.ParseMultipartForm(_MAX_UPLOAD_MEM)
  if err != nil { serveBadRequestHX(w, err.Error()); return }

  name, err := bind.FormValue(r, "name", "required")
  if err != nil { serveBadRequestHX(w, err.Error()); return }

  priceStr, err := bind.FormValue(r, "price", "required,numeric")
  if err != nil { serveBadRequestHX(w, err.Error()); return }
  price, err := strconv.ParseFloat(priceStr, 32)
  if err != nil { serveBadRequestHX(w, err.Error()); return }

  desc := r.FormValue("desc")

  imgPath := ""
  file, header, err := r.FormFile("img")
  if err == nil {
    filename, err := upload.SaveImage(file, header, static.IMGDIR)
    if err != nil { serveBadRequestHX(w, err.Error()); return }
    imgPath = "/" + static.IMGDIR + "/" + filename
  } else if err != http.ErrMissingFile {
    serveBadRequestHX(w, err.Error()); return
  }

  item := models.Item{
    Name:      name,
    Price:     float32(price),
    TimeMod:   time.Now().Unix(),
    Available: true,
    Desc:      desc,
    ImgPath:   imgPath,
  }
  err = models.InsertItem(item)
  if err != nil { serveInternalErrHX(w); return }

  serveResponseHX(w, "Se creó el item", Created, &_NextAction{
    URL: "htmx/form-create-item",
    Name: "Crear otro",
    HTMX: true,
  })
}

func GetItems(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { serveBadRequest(w, r, _ErrBadSearch); return }

  var data _MainData

  data.Data, err = models.GetItems(params)
  if err != nil{ serveInternalErr(w, r); return }
  if len(data.Data.([]models.Item)) <= 0 { data.Msg = _MsgEmpty }

  serveResponse(w, r, &data, OK, nil)
}

func GetItemFromID(w http.ResponseWriter, r *http.Request) {
  idStr := chi.URLParam(r, models.ITEM_ID)
  id, err := strconv.Atoi(idStr)
  if err != nil { serveNoResults(w, r); return }

  item, err := models.GetItemFromID(id)
  if err != nil { serveInternalErr(w, r); return }

  if item.Name == "" { serveNoResults(w, r); return }

  serveData(w, r, item)
}

func GetItemEditForm(w http.ResponseWriter, r *http.Request) {
  idStr := chi.URLParam(r, models.ITEM_ID)
  id, err := strconv.Atoi(idStr)
  if err != nil { serveBadRequestHX(w, "ID inválido"); return }

  item, err := models.GetItemFromID(id)
  if err != nil { serveInternalErrHX(w); return }
  if item.Name == "" { serveBadRequestHX(w, "Ítem no encontrado"); return }

  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  if err := _Tpl.ExecuteTemplate(w, "form-edit-item", item); err != nil {
    slog.Error(err.Error())
    serveInternalErrHX(w)
  }
}

func UpdateItem(w http.ResponseWriter, r *http.Request) {
  idStr := chi.URLParam(r, models.ITEM_ID)
  id, err := strconv.Atoi(idStr)
  if err != nil { serveBadRequestHX(w, "ID inválido"); return }

  update := models.ItemUpdate{
    Name: r.FormValue("name"),
    Desc: r.FormValue("desc"),
  }

  if priceStr := r.FormValue("price"); priceStr != "" {
    price, err := strconv.ParseFloat(priceStr, 32)
    if err != nil { serveBadRequestHX(w, "Precio inválido"); return }
    p := float32(price)
    update.Price = &p
  }

  if availStr := r.FormValue("available"); availStr != "" {
    avail := availStr == "true"
    update.Available = &avail
  }

  err = models.UpdateItem(id, update)
  if err != nil { serveInternalErrHX(w); return }

  serveResponseHX(w, "Se actualizó el ítem", OK, nil)
}
