package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
  "github.com/ajderniz/repostele/pkg/upload"
  "github.com/ajderniz/repostele/static"
)

const (
  _ITEM_ID = "id"
  _MAX_UPLOAD_MEM = 10 << 20 // 10 MiB, incl. non-file form fields
)

func PostItem(w http.ResponseWriter, r *http.Request) {
  err := r.ParseMultipartForm(_MAX_UPLOAD_MEM)
  if err != nil { serveBadRequest(w, r, err); return }

  name, err := bind.FormValue(r, "name", "required")
  if err != nil { serveBadRequest(w, r, err); return }

  priceStr, err := bind.FormValue(r, "price", "required,numeric")
  if err != nil { serveBadRequest(w, r, err); return }
  price, err := strconv.ParseFloat(priceStr, 32)
  if err != nil { serveBadRequest(w, r, err); return }

  desc := r.FormValue("desc")

  imgPath := ""
  file, header, err := r.FormFile("img")
  if err == nil {
    filename, err := upload.SaveImage(file, header, static.IMGDIR)
    if err != nil { serveBadRequest(w, r, err); return }
    imgPath = "/" + static.IMGDIR + "/" + filename
  } else if err != http.ErrMissingFile {
    serveBadRequest(w, r, err); return
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
  if err != nil { serveInternalErr(w, r); return }

  serveResponse(
    w, r, &_MainData{Msg: "Item posted succesfully"}, Created, nil,
  )
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

func UpdateItem(w http.ResponseWriter, r *http.Request) {
  idStr, err := bind.FormValue(r, _ITEM_ID, "required,number")
  if err != nil { serveResponse(w, r, nil, BadRequest, err); return }
  id, _ := strconv.Atoi(idStr)

  update := models.ItemUpdate{}
  err = bind.JSON(r, &update)
  if err != nil { serveResponse(w, r, nil, BadRequest, err); return }

  err = models.UpdateItem(id, update)
  if err != nil {
    serveResponse(w, r, nil, InternalServerError, err)
    return
  }

  serveMsg(w, r, "Item updated successfully")
}
