package models

import (
	"github.com/ajderniz/repostele/pkg/errman"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var _DB *sqlx.DB

func OpenDB() error {
	db, err := sqlx.Open("sqlite3", "assets/data.db")
	errman.CheckFatal(err)
	_DB = db
	return nil
}
