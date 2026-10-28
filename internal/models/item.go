package models

import (
	"errors"
	"reflect"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
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

var _ItemSortFields = []string { ITEM_ID, _ITEM_NAME, _ITEM_PRICE, _ITEM_TIME_MOD }

func InsertItem(item Item) (int, error) {
  result, err := dbNamedExec(
    "INSERT INTO "+_ITEMS+" ("+_ITEM_FIELDS+") "+
    "VALUES "+"(:"+_ITEM_NAME+",:"+_ITEM_PRICE+",:"+_ITEM_TIME_MOD+",:"+
      _ITEM_AVAILABLE+",:"+_ITEM_DESC+",:"+_ITEM_IMG_PATH+")",
    &item,
  )
  if err != nil { return 0, dbErr(errors.New("No se pudo publicar el nuevo ítem"), err) }
  id, err := result.LastInsertId()
  if err != nil { return 0, dbErr(errors.New("No se pudo publicar el nuevo ítem"), err) }
  return int(id), nil
}

// GetItems pages through the menu; unavailable items are left out unless includeUnavailable.
func GetItems(params *SelectParams, includeUnavailable bool) ([]Item, bool, error) {
  query := "SELECT * FROM "+_ITEMS
  if !includeUnavailable { query += " WHERE "+_ITEM_AVAILABLE+" = 1" }
  items, hasNext, err := selectPage[Item](query, params, _ItemSortFields)
  if err != nil {
    return []Item{}, false, dbErr(errors.New("No se pudo acceder a la lista de ítemes"), err)
  }
  return items, hasNext, nil
}

// CartItem is what the cart page needs to refresh a stored cart.
type CartItem struct {
  Id    int     `db:"id"    json:"id"`
  Name  string  `db:"name"  json:"name"`
  Price float32 `db:"price" json:"price"`
}

func GetAvailableCartItems() ([]CartItem, error) {
  items := []CartItem{}
  err := dbSelect(&items,
    "SELECT "+ITEM_ID+","+_ITEM_NAME+","+_ITEM_PRICE+" FROM "+_ITEMS+" "+
    "WHERE "+_ITEM_AVAILABLE+" = 1",
  )
  if err != nil { return []CartItem{}, dbErr(errors.New("No se pudo acceder a la lista de ítemes"), err) }
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
  if err != nil { return Item{}, dbErr(errors.New("No se pudo acceder al ítem"), err) }
  return item, nil
}

// GetItemsFromIDs loads several items in one query. Unknown ids are absent from the map.
func GetItemsFromIDs(ids []int) (map[int]Item, error) {
  items := map[int]Item{}
  if len(ids) == 0 { return items, nil }

  query, args, err := sqlx.In(
    "SELECT * FROM "+_ITEMS+" WHERE "+ITEM_ID+" IN (?)", ids,
  )
  if err != nil { return nil, dbErr(errors.New("No se pudo acceder a los ítemes"), err) }

  list := []Item{}
  err = dbSelect(&list, query, args...)
  if err != nil { return nil, dbErr(errors.New("No se pudo acceder a los ítemes"), err) }
  for _, it := range list { items[it.Id] = it }
  return items, nil
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
    var val any
    switch field.Kind() {
    case reflect.Pointer:
      if field.IsNil() { continue }
      val = field.Elem().Interface()
    case reflect.String:
      if field.String() == "" { continue }
      val = field.Interface()
    default:
      val = field.Interface()
    }

    setClauses = append(setClauses, dbTag+" = ?")
    args = append(args, val)
  }
  args = append(args, id)

  _, err := dbExec(
    "UPDATE "+_ITEMS+" "+
    "SET "+strings.Join(setClauses,",")+" "+
    "WHERE "+ITEM_ID+" = ?",
    args...,
  )
  if err != nil {
    return dbErr(errors.New("No se pudo actualizar el ítem"), err)
  }

  return nil
}

