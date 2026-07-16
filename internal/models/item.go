package models

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/ajderniz/repostele/pkg/errman"
	_ "github.com/mattn/go-sqlite3"
)

type Item struct {
  Id      int    `db:"id"       schema:"id"       json:"id"       validate:"-"`
  Name    string `db:"name"     schema:"name"     json:"name"     validate:"required"`
  Price   int    `db:"price"    schema:"price"    json:"price"    validate:"required,gte=0"`
  TimeMod int64  `db:"time_mod" schema:"time_mod" json:"time_mod" validate:"required,gte=0"`
  Desc    string `db:"desc"     schema:"desc"     json:"desc"     validate:"-"`
  ImgPath string `db:"img_path" schema:"img_path" json:"img_path" validate:"-"`
}

const (
  _ITEMS       = "items"
  _ITEM_ID     = "id"
  _ITEM_NAME   = "name"
  _ITEM_PRICE  = "price"
)

var _SortFields = map[string]bool { 
  _ITEM_ID:    true,
  _ITEM_NAME:  true,
  _ITEM_PRICE: true,
  "mod":       true,
}

const (
  _SORT_DIR_ASC  string = "ASC"
  _SORT_DIR_DESC string = "DESC"
)

func GetItems(start int, limit int, sort string, dir string) ([]Item, error) {
  if start < 0 { start = 0 }
  if _, exists := _SortFields[sort]; !exists { sort = _ITEM_ID }
  dir = strings.ToUpper(dir)
  if dir != _SORT_DIR_ASC && dir != _SORT_DIR_DESC { dir = _SORT_DIR_ASC }

  items := []Item{}
  err := _DB.Select(&items,
    "SELECT * "+
    "FROM "+_ITEMS+" "+
    "ORDER BY "+sort+" "+dir+" "+
    "LIMIT "+strconv.Itoa(start)+","+strconv.Itoa(limit),
  )
  if err != nil {
    errman.PrintError(err)
    return nil, errors.New("Could not retreive items")
  } 

  return items, err
}

func GetItemFromId(id string) (Item, error) {
  item := Item{}
  err := _DB.Get(&item,
    "SELECT * "+
    "FROM "+_ITEMS+" "+
    "WHERE "+_ITEM_ID+" = $1",
    id,
  )
  if err != nil { 
    errman.PrintError(err)
    if err == sql.ErrNoRows { return Item{}, nil }
    return Item{}, errors.New("Could not retreive item")
  }
  return item, nil
}