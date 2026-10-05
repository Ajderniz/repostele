package models

import (
	"database/sql"
	_ "embed"
	"errors"
	"log/slog"
	"slices"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const _DB_FILEPATH = "./data.db"

// Indexes for the hot lookups (latest order per user, queue by status, order items, user sessions).
// Idempotent, so it runs on every start and also covers DBs created before they existed.
var _INDEXES = []string{
  "CREATE INDEX IF NOT EXISTS idx_orders_user_time ON orders(user, time)",
  "CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status)",
  "CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id)",
  "CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user)",
}

func ensureIndexes() error {
  for _, q := range _INDEXES {
    if _, err := _DB.Exec(q); err != nil { return err }
  }
  return nil
}

// PurgeExpired drops sessions and fingerprints whose expiry has passed.
// Expired fingerprints are already ignored by lookups (see GetFingerPrintFromID), so this is safe.
func PurgeExpired(now int64) error {
  _, err := _DB.Exec(
    "DELETE FROM "+_SESSIONS+" WHERE "+_SESSION_EXPIRES+" <= ?", now,
  )
  if err != nil { return err }
  _, err = _DB.Exec(
    "DELETE FROM "+_FINGERPRINTS+" WHERE "+_FINGERPRINT_EXPIRES+" <= ?", now,
  )
  return err
}

var (
  _DB *sqlx.DB

  _ErrInsertAcc = errors.New("No se pudo crear la cuenta")
  _ErrGetAccs = errors.New("No se pudo acceder a la lista de cuentas")
  _ErrGetAcc = errors.New("No se pudo acceder a la cuenta")
  _ErrModAcc = errors.New("No se pudo modificar la cuenta")
  _ErrCloseSession = errors.New("No se pudo cerrar la sesión")
)

type SortDir string
const (
  SORT_DIR_ASC  SortDir = "ASC"
  SORT_DIR_DESC SortDir = "DESC"
)

type _SortFields []string

type SelectParams struct {
  Start int     `schema:"start,default:0"`
  Limit int     `schema:"limit,default:10"`
  Sort  string  `schema:"sort,default:id"`
  Dir   SortDir `schema:"dir"`
}

func (params *SelectParams) Fix(fields _SortFields) {
  if params.Start < 0 { params.Start = 0 }
  if params.Limit <= 0 { params.Limit = 99 }
  if !slices.Contains(fields, params.Sort) { params.Sort = fields[0] }
  if params.Dir != SORT_DIR_ASC && params.Dir != SORT_DIR_DESC { 
    params.Dir = SORT_DIR_ASC 
  }
}

func dbSelectErr(err error) error {
  if err != nil {
    if err == sql.ErrNoRows{ return nil }
    slog.Error(err.Error())
    return err
  }
  return nil
}

func dbGet(dst any, query string, args ...any) error {
  err := _DB.Get(dst, query, args...)
  return dbSelectErr(err)
}

func dbSelect(dst any, query string, args ...any) error {
  err := _DB.Select(dst, query, args...)
  return dbSelectErr(err)
}

func dbBeginExecAndCommit(query string, args ...any) (sql.Result, error) {
  tx, err := _DB.Beginx()
  if err != nil { slog.Error(err.Error()); return nil, err }

  result, err := tx.Exec(query, args...)
  if err != nil { tx.Rollback(); slog.Error(err.Error()); return nil, err }

  err = tx.Commit()
  if err != nil { tx.Rollback(); slog.Error(err.Error()); return nil, err }

  return result, nil
}

func dbBeginNamedExecAndCommit(query string, v any) (sql.Result, error) {
  tx, err := _DB.Beginx()
  if err != nil { slog.Error(err.Error()); return nil, err }

  result, err := tx.NamedExec(query, v)
  if err != nil { tx.Rollback(); slog.Error(err.Error()); return nil, err }

  err = tx.Commit()
  if err != nil { tx.Rollback(); slog.Error(err.Error()); return nil, err }

  return result, nil
}

func dbUpdateTableField(
  table, set string, 
  v any, 
  where string, 
  eq any,
) (result sql.Result, err error) {

  result, err = dbBeginExecAndCommit(
    "UPDATE "+table+" "+
    "SET "+set+" = ? "+
    "WHERE "+where+" = ?",
    v, eq,
  )
  return
}

func dbSelectList(
  dst any,
  sel, from string,
  params *SelectParams,
  sortFields _SortFields,
) error {

  params.Fix(sortFields)
  err := dbSelect(dst,
    "SELECT "+sel+" FROM "+from+" ORDER BY "+params.Sort+" "+string(params.Dir)+
    " LIMIT ?, ?",
    params.Start, params.Limit,
  )
  if dbSelectErr(err) != nil {
    slog.Error(err.Error())
    return errors.New("Error de selección")
  }
  return nil
}
