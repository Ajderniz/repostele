package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
	"github.com/ajderniz/repostele/pkg/write"
)

type _GetItemsQuery struct {
  Start int    `schema:"start,default:0"`
  Limit int    `schema:"limit,default:10"`
  Sort  string `schema:"sort,default:id"`
  Dir   string `schema:"order,default:ASC"`
}

func GetItems(w http.ResponseWriter, r *http.Request) {
  query := _GetItemsQuery{}
  err := bind.Form(r, &query)
  if err != nil {
    write.ErrorJSON(w, http.StatusBadRequest, err)
    return
  }

  items, err := models.GetItems(
    query.Start, query.Limit, query.Sort, query.Dir,
  )
  if err != nil {
    write.ErrorJSON(w, http.StatusInternalServerError, err)
    return
  }
  if items == nil {
    write.JSON(w, http.StatusOK, write.H{"data": "No records"})
    return
  }

  write.JSON(w, http.StatusOK, write.H{"data": items})
}

func GetItemFromId(w http.ResponseWriter, r *http.Request) {
  id := chi.URLParam(r, "id")
  if _, err := strconv.Atoi(id); err != nil { 
    write.ErrorJSON(w, http.StatusNotAcceptable, errors.New("Bad search criteria"))
    return
  }

  item, err := models.GetItemFromId(id)
  if err != nil {
    write.ErrorJSON(w, http.StatusInternalServerError, err)
    return
  }

  if item.Name == "" {
    write.JSON(w, http.StatusOK, write.H{"data": "Not found"})
    return
  }

  write.JSON(w, http.StatusOK, write.H{"data": item})
}