package controllers

import (
	"net/http"
  "os"
  "path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajderniz/repostele/internal/models"

	"github.com/ajderniz/repostele/internal/reqlog"
	"github.com/ajderniz/repostele/pkg/bind"
  "github.com/ajderniz/repostele/pkg/upload"
  "github.com/ajderniz/repostele/static"
)

const (
  _MAX_UPLOAD_MEM = 10 << 20 // 10 MiB, incl. non-file form fields
)

func GetItemCreateForm(w http.ResponseWriter, r *http.Request) {
  if !isHX(r) { http.NotFound(w, r); return }
  onLastPage := r.URL.Query().Get("hasNext") != "true"
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  _Tpl.ExecuteTemplate(w, "form-create-item",
    map[string]any{"OnLastPage": onLastPage})
}

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
  id, err := models.InsertItem(item)
  if err != nil { serveInternalErrHX(w, r, err); return }
  item.Id = id

  serveOOB(w, Created, _HXData{Msg: "Se creó el item", NextAction: &_NextAction{
    URL: "htmx/form-create-item",
    Name: "Crear otro",
    HTMX: true,
  }})

  onLastPage := r.FormValue("on_last_page") == "true"
  if onLastPage {
    w.Write([]byte(`<p id="menu-empty-msg" hx-swap-oob="delete"></p>`))
    w.Write([]byte(`<div hx-swap-oob="beforeend:#menu-item-list">`))
    _Tpl.ExecuteTemplate(w, "menu-item", map[string]any{
      "Item": item, "LoggedIn": true, "IsStaff": true, "IsAdmin": true,
      "OOB": "",
    })
    w.Write([]byte(`</div>`))
  }
}

func GetItems(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { serveBadRequest(w, r, _ErrBadSearch); return }

  var data _MainData

  // Only admins see (and can re-enable) unavailable items.
  isAdmin := false
  session, hasSession := models.SessionFromContext(r.Context())
  if hasSession && session.Role.IsStaff() {
    staff, err := models.GetStaffFromUsername(session.User)
    if err != nil { serveInternalErr(w, r, err); return }
    isAdmin = staff.Admin
  }

  items, hasNext, err := models.GetItems(&params, isAdmin)
  if err != nil{ serveInternalErr(w, r, err); return }
  data.Data = items
  data.Params = params
  data.HasNext = hasNext
  if len(items) <= 0 { data.Msg = _MsgEmpty }

  if hasSession && session.Role == models.SESSION_ROLE_USER {
    order, err := models.GetLatestOrderFromUsername(session.User)
    if err != nil {
      reqlog.Error(r, err)
    } else if order.Pending() {
      data.CurrentOrder = &order
    }
  }

  serveResponse(w, r, &data, OK, nil)
}

func GetItemEditForm(w http.ResponseWriter, r *http.Request) {
  idStr := chi.URLParam(r, models.ITEM_ID)
  id, err := strconv.Atoi(idStr)
  if err != nil { serveBadRequestHX(w, "ID inválido"); return }

  item, err := models.GetItemFromID(id)
  if err != nil { serveInternalErrHX(w, r, err); return }
  if item.Name == "" { serveBadRequestHX(w, "Ítem no encontrado"); return }

  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  if err := _Tpl.ExecuteTemplate(w, "form-edit-item", item); err != nil {
    serveInternalErrHX(w, r, err)
  }
}

func UpdateItem(w http.ResponseWriter, r *http.Request) {
  idStr := chi.URLParam(r, models.ITEM_ID)
  id, err := strconv.Atoi(idStr)
  if err != nil { serveBadRequestHX(w, "ID inválido"); return }

  err = r.ParseMultipartForm(_MAX_UPLOAD_MEM)
  if err != nil { serveBadRequestHX(w, err.Error()); return }

  oldItem, err := models.GetItemFromID(id)
  if err != nil { serveInternalErrHX(w, r, err); return }

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

  file, header, err := r.FormFile("img")
  if err == nil {
    filename, err := upload.SaveImage(file, header, static.IMGDIR)
    if err != nil { serveBadRequestHX(w, err.Error()); return }
    update.ImgPath = "/" + static.IMGDIR + "/" + filename
  } else if err != http.ErrMissingFile {
    serveBadRequestHX(w, err.Error()); return
  }

  err = models.UpdateItem(id, update)
  if err != nil { serveInternalErrHX(w, r, err); return }

  if update.ImgPath != "" && oldItem.ImgPath != "" {
    old := filepath.Join(static.IMGDIR, filepath.Base(oldItem.ImgPath))
    if err := os.Remove(old); err != nil {
      reqlog.Error(r, err)
    }
  }

  item, err := models.GetItemFromID(id)
  if err != nil { serveInternalErrHX(w, r, err); return }

  serveOOB(w, OK, _HXData{Msg: "Se actualizó el ítem"},
    frag("menu-item", map[string]any{
      "Item": item, "LoggedIn": true, "IsStaff": true, "IsAdmin": true,
      "OOB": true,
    }))
}
