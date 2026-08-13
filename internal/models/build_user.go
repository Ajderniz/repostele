//go:build USER

package models

import (
	"os"

	"github.com/jmoiron/sqlx"
)

func OpenDB() error {
  db, err := sqlx.Open("sqlite3", _DB_FILEPATH)
  if err != nil { return err }
  _DB = db

  _, err = os.Stat(_DB_FILEPATH)
  if err != nil { return err }

  return nil
}
