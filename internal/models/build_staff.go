//go:build STAFF

package models

import (
	"errors"
	"os"
	"strings"

	"github.com/ajderniz/repostele/static"
	"github.com/jmoiron/sqlx"
)

func OpenDB() error {
  db, err := sqlx.Open("sqlite3", _DB_FILEPATH)
  if err != nil { return err }
  _DB = db

  _, err = os.Stat(_DB_FILEPATH)
  if err != nil {
    if errors.Is(err, os.ErrNotExist) {

      schemaStr, err := static.FS.ReadFile("schema.sql")
      if err != nil { return err }

      tx, err := _DB.Beginx()
      if err != nil { return err }

      for s := range strings.SplitSeq(string(schemaStr), ";") {
        _, err = _DB.Exec(s)
        if err != nil { tx.Rollback(); return err }
      }

      err = tx.Commit()
      if err != nil { tx.Rollback(); return err }

    } else {
      return err
    }
  }

  return nil
}
