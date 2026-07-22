package models

import (
	"database/sql"
	"errors"

	"github.com/ajderniz/repostele/pkg/errman"
)

type User struct {
  Username     string `db:"username"`
  PassHash     string `db:"pass_hash"`
  TimeCreated  int64  `db:"time_created"`
  Active       bool   `db:"active"`
}

const (
  _USERS             = "users"
  USER_USERNAME      = "username"
  _USER_PASS_HASH    = "pass_hash"
  _USER_TIME_CREATED = "time_created"
  _USER_ACTIVE       = "active"
  _USER_FIELDS       = USER_USERNAME+","+_USER_PASS_HASH+","+_USER_TIME_CREATED+
                       ","+_USER_ACTIVE
)

var _RegisterErr = errors.New("Could not create user")

func InsertUser(user User) error {
  tx, err := _DB.Beginx()
  if err != nil { errman.PrintError(err); return _RegisterErr }

  _, err = tx.NamedExec(
    "INSERT INTO "+_USERS+" ("+_USER_FIELDS+") " +
    "VALUES (:"+USER_USERNAME+",:"+_USER_PASS_HASH+",:"+_USER_TIME_CREATED+")",
    &user,
  )
  if err != nil { errman.PrintError(err); return _RegisterErr }
  
  err = tx.Commit()
  if err != nil { errman.PrintError(err); return _RegisterErr }

  return nil
}

func GetUserFromUsername(username string) (User, error) {
  user := User{}
  err := _DB.Get(&user,
    "SELECT * "+
    "FROM "+_USERS+" "+
    "WHERE "+USER_USERNAME+" = $1 AND "+_USER_ACTIVE+" = true",
    username,
  )
  if err != nil { 
    errman.PrintError(err)
    if err == sql.ErrNoRows { return User{}, nil }
    return User{}, errors.New("Could not retreive user")
  }
  return user, nil
}

var _DeleteErr = errors.New("Could not delete user")

func DeactivateUser(username string) error {
  tx, err := _DB.Beginx()
  if err != nil { errman.PrintError(err); return _DeleteErr }

  _, err = tx.Exec(
    "UPDATE "+_USERS+" "+
    "SET "+_USER_ACTIVE+" = false "+
    "WHERE "+USER_USERNAME+" = $1",
    username,
  )
  if err != nil { errman.PrintError(err); return _DeleteErr }

  err = tx.Commit()
  if err != nil { errman.PrintError(err); return _DeleteErr }

  return nil
}
