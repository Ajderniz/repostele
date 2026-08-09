package models

import (
	"errors"
	"reflect"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/ajderniz/repostele/pkg/errman"
)

type Item struct {
  Id        int     `db:"id"        schema:"id"        json:"id"        validate:"-"`
  Name      string  `db:"name"      schema:"name"      json:"name"      validate:"required"`
  Price     float32 `db:"price"     schema:"price"     json:"price"     validate:"required,gte=0"`
  TimeMod   int64   `db:"time_mod"  schema:"time_mod"  json:"time_mod"  validate:"required,gte=0"`
  Available bool    `db:"available" schema:"available" json:"available" validate:"required"`
  Desc      string  `db:"desc"      schema:"desc"      json:"desc"      validate:"-"`
  ImgPath   string  `db:"img_path"  schema:"img_path"  json:"img_path"  validate:"-"`
}

const (
  _ITEMS          = "items"
  ITEM_ID         = "id"
  _ITEM_NAME      = "name"
  _ITEM_PRICE     = "price"
  _ITEM_TIME_MOD  = "time_mod"
  _ITEM_AVAILABLE = "available"
  _ITEM_DESC      = "desc"
  _ITEM_IMG_PATH  = "img_path"
  _ITEM_FIELDS = _ITEM_NAME+","+_ITEM_PRICE+","+_ITEM_TIME_MOD+","+
                 _ITEM_AVAILABLE+","+_ITEM_DESC+","+_ITEM_IMG_PATH
)

var _ItemSortFields = []string { ITEM_ID, _ITEM_NAME, _ITEM_PRICE, "mod" }

func InsertItem(item Item) error {
  _, err := dbBeginNamedExecAndCommit(
    "INSERT INTO "+_ITEMS+" ("+_ITEM_FIELDS+") "+
    "VALUES "+"(:"+_ITEM_NAME+",:"+_ITEM_PRICE+",:"+_ITEM_TIME_MOD+",:"+
      _ITEM_AVAILABLE+",:"+_ITEM_DESC+",:"+_ITEM_IMG_PATH+")",
    &item,
  )
  if err != nil { return errors.New("Could not post new item") }
  return nil
}

func GetItems(params SelectParams) ([]Item, error) {
  items := []Item{}
  err := dbSelectList(&items, "*", _ITEMS, params, _ItemSortFields)
  if err != nil { return []Item{}, errors.New("Could not retrieve item list") }
  return items, nil
}

func GetItemFromID(id int) (Item, error) {
  item := Item{}
  err := dbGet(&item,
    "SELECT * "+
    "FROM "+_ITEMS+" "+
    "WHERE "+ITEM_ID+" = ?",
    id,
  )
  if err != nil { return Item{}, errors.New("Could not retrieve item") }
  return item, nil
}

type ItemUpdate struct {
    Name      string   `db:"name"      json:"name"`
    Price     *float32 `db:"price"     json:"price"    validate:"gte=0"`
    Available *bool    `db:"available" json:"available"`
    Desc      string   `db:"desc"      json:"desc"`
    ImgPath   string   `db:"img_path"  json:"img_path"`
}

func UpdateItem(id int, update ItemUpdate) error {
  var setClauses []string
  var args []any

  v := reflect.ValueOf(update)
  t := reflect.TypeFor[ItemUpdate]()
  for i := 0; i < v.NumField(); i++ {
    dbTag := t.Field(i).Tag.Get("db")
    if dbTag == "" || dbTag == "-" { continue }

    field := v.Field(i)
    switch field.Kind() {
    case reflect.Pointer: if field.IsNil() { continue }
    case reflect.String:  if field.String() == "" { continue }
    }

    setClauses = append(setClauses, dbTag+" = ?")
    args = append(args, field.Elem().Interface())
  }
  args = append(args, id)

  _, err := dbBeginExecAndCommit(
    "UPDATE "+_ITEMS+" "+
    "SET "+strings.Join(setClauses,",")+" "+
    "WHERE "+ITEM_ID+" = ?",
    args,
  )
  if err != nil {
    errman.PrintError(err)
    return errors.New("Could not update item")
  }

  return nil
}

