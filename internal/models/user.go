package models

import (
	"database/sql"
	"errors"
	"time"

	"github.com/ajderniz/repostele/pkg/errman"
	"github.com/ajderniz/repostele/pkg/pass"
)

type UserInfo struct {
  Username    string `json:"username"`
  TimeCreated int64  `json:"time_created"`
}

type User struct {
  Username     string `db:"username"`
  PassHash     string `db:"pass_hash"`
  TimeCreated  int64  `db:"time_created"`
}

const (
  _USERS             = "users"
  USER_USERNAME      = "username"
  _USER_PASS_HASH    = "pass_hash"
  _USER_TIME_CREATED = "time_created"
  _USER_FIELDS       = USER_USERNAME+","+_USER_PASS_HASH+","+_USER_TIME_CREATED
)

var _RegisterErr = errors.New("Could not create user")

func RegisterUser(username, password string) error {
	user := User{}
	var err error

	user.Username = username
	user.PassHash, err = pass.HashPassword(password)
	if err != nil { return err }
  user.TimeCreated = time.Now().Unix()

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
    "WHERE "+USER_USERNAME+" = $1",
    username,
  )
  if err != nil { 
    errman.PrintError(err)
    if err == sql.ErrNoRows { return User{}, errors.New("User not found") }
    return User{}, errors.New("Could not retreive user")
  }
  return user, nil
}

func UserLogin(username, password string) (UserInfo, error) {
	user, err := GetUserFromUsername(username)
	if err != nil { return UserInfo{}, err }
	if err := pass.CheckPasswordHash(password, user.PassHash); err != nil {
		return UserInfo{}, err
	}
	return UserInfo{ user.Username, user.TimeCreated }, nil
}
