package models

import (
	"database/sql"
	_ "embed"
	"errors"
	"log/slog"
	"slices"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const _DB_FILEPATH = "./data.db"

// WAL lets readers (the polls) run while the other app writes; the busy
// timeout makes a writer wait for the lock instead of failing.
const _DB_DSN = _DB_FILEPATH+"?_journal_mode=WAL&_busy_timeout=5000"

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

// Columns added after the first schema. Each is added, then backfilled, in one
// transaction on DBs that lack it. Rows that predate the order_items snapshot
// columns get the item's current values, the best guess left.
var _ADDED_COLUMNS = []struct{ table, column, def, backfill string }{
  {"order_items", "price", "NUMERIC CHECK(0 <= price)",
    "UPDATE order_items SET price = (SELECT price FROM items WHERE items.id = order_items.item_id) WHERE price IS NULL"},
  {"order_items", "name", "TEXT",
    "UPDATE order_items SET name = (SELECT name FROM items WHERE items.id = order_items.item_id) WHERE name IS NULL"},
}

func ensureColumns() error {
  for _, c := range _ADDED_COLUMNS {
    var n int
    err := _DB.Get(&n, "SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?", c.table, c.column)
    if err != nil { return err }
    if n > 0 { continue }

    tx, err := _DB.Beginx()
    if err != nil { return err }
    _, err = tx.Exec("ALTER TABLE "+c.table+" ADD COLUMN "+c.column+" "+c.def)
    if err != nil {
      tx.Rollback()
      // The other app may have migrated between the check and the ALTER.
      if strings.Contains(err.Error(), "duplicate column") { continue }
      return err
    }
    if _, err = tx.Exec(c.backfill); err != nil { tx.Rollback(); return err }
    if err = tx.Commit(); err != nil { return err }
  }
  return nil
}

// ensureSchema brings an existing DB up to date; safe to run on every start.
func ensureSchema() error {
  if err := ensureColumns(); err != nil { return err }
  return ensureIndexes()
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

// Single statements run in SQLite's implicit transaction.
func dbExec(query string, args ...any) (sql.Result, error) {
  result, err := _DB.Exec(query, args...)
  if err != nil { slog.Error(err.Error()); return nil, err }
  return result, nil
}

func dbNamedExec(query string, v any) (sql.Result, error) {
  result, err := _DB.NamedExec(query, v)
  if err != nil { slog.Error(err.Error()); return nil, err }
  return result, nil
}

func dbUpdateTableField(
  table, set string, 
  v any, 
  where string, 
  eq any,
) (result sql.Result, err error) {

  result, err = dbExec(
    "UPDATE "+table+" "+
    "SET "+set+" = ? "+
    "WHERE "+where+" = ?",
    v, eq,
  )
  return
}

// selectPage runs query (a SELECT with no ORDER BY or LIMIT) for the page in
// params. It fetches one row past the page, so hasNext is exact.
func selectPage[T any](
  query string,
  params *SelectParams,
  sortFields _SortFields,
  args ...any,
) (rows []T, hasNext bool, err error) {

  params.Fix(sortFields)
  rows = []T{}
  err = dbSelect(&rows,
    query+" ORDER BY "+params.Sort+" "+string(params.Dir)+" LIMIT ?, ?",
    append(args, params.Start, params.Limit+1)...,
  )
  if err != nil { return []T{}, false, err }
  if len(rows) > params.Limit { return rows[:params.Limit], true, nil }
  return rows, false, nil
}
