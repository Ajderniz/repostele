package models

import (
	"database/sql"
	_ "embed"
	"errors"
	"os"
	"slices"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/ajderniz/repostele/pkg/errman"
)

const _DB_FILEPATH = "./assets/data.db"

var (
  _DB *sqlx.DB

  //go:embed schema.sql
  _Schema string

  _ErrInsertAcc = errors.New("Could not create account")
  _ErrGetAccs = errors.New("Could not retrieve accout list")
  _ErrGetAcc = errors.New("Could not retrieve accout")
  _ErrModAcc = errors.New("Could not modify account")
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
  Dir   SortDir `schema:"dir,deafult:asc"`
}

func (params SelectParams) Fix(fields _SortFields) {
  if params.Start < 0 { params.Start = 0 }
  if params.Limit <= 0 { params.Limit = 99 }
  if !slices.Contains(fields, params.Sort) { params.Sort = fields[0] }
  if params.Dir != SORT_DIR_ASC && params.Dir != SORT_DIR_DESC { params.Dir = SORT_DIR_ASC }
}

func dbSelectErr(err error) error {
  if err != nil {
    if err == sql.ErrNoRows{ return nil }
    errman.PrintError(err)
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
  if err != nil { errman.PrintError(err); return nil, err }

  result, err := tx.Exec(query, args...)
  if err != nil { errman.PrintError(err); return nil, err }

  err = tx.Commit()
  if err != nil { errman.PrintError(err); return nil, err }

  return result, nil
}

func dbBeginNamedExecAndCommit(query string, v any) (sql.Result, error) {
  tx, err := _DB.Beginx()
  if err != nil { errman.PrintError(err); return nil, err }

  result, err := tx.NamedExec(query, v)
  if err != nil { errman.PrintError(err); return nil, err }

  err = tx.Commit()
  if err != nil { errman.PrintError(err); return nil, err }

  return result, nil
}

func dbUpdateTableField(table, set string, v any, where string, eq any) (result sql.Result, err error) {
  result, err = dbBeginExecAndCommit(
    "UPDATE "+table+" "+
    "SET "+set+" = ? "+
    "WHERE "+where+" = ?",
    v, eq,
  )
  return
}

func dbSelectList(dst any, sel, from string, params SelectParams, sortFields _SortFields) error {
  params.Fix(sortFields)
  err := dbSelect(dst,
    "SELECT "+sel+" FROM "+from+" ORDER BY ? "+string(params.Dir)+" LIMIT ?, ?",
    params.Sort, params.Start, params.Limit,
  )
  if dbSelectErr(err) != nil {
    errman.PrintError(err)
    return errors.New("Selection error")
  }
  return nil
}

func OpenDB() error {
  db, err := sqlx.Open("sqlite3", _DB_FILEPATH)
  if err != nil { return err }
  _DB = db

  _, err = os.Stat(_DB_FILEPATH)
  if err != nil {
    if errors.Is(err, os.ErrNotExist) {

      schema := strings.Split(_Schema, ";")

      tx, err := _DB.Beginx()
      if err != nil { return err }

      for _, s := range schema {
        _, err = _DB.Exec(s)
        if err != nil { return err }
      }

      err = tx.Commit()
      if err != nil { return err }

    } else { 
      return err 
    }
  }

  return nil
}
