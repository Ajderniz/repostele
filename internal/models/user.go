package models

import (
	"github.com/ajderniz/repostele/pkg/errman"
)

type User struct {
  Username     string `db:"username"     json:"username"`
  PassHash     string `db:"pass_hash"`
  TimeCreated  int64  `db:"time_created" json:"time_created"`
  Active       bool   `db:"active"       json:"active"`
}

const (
  _USERS            = "users"
  USER_USERNAME     = "username"
  USER_PASS_HASH    = "pass_hash"
  USER_TIME_CREATED = "time_created"
  USER_ACTIVE       = "active"
  _USER_FIELDS       = USER_USERNAME+","+/*_USER_PASS_HASH+","+*/USER_TIME_CREATED+
                       ","+USER_ACTIVE
)

func InsertUserAccount(user User, fp Fingerprint) error {
  tx, err := _DB.Beginx()
  if err != nil { errman.PrintError(err); return _ErrInsertAcc }

  _, err = tx.NamedExec(
    "INSERT INTO "+_USERS+" ("+USER_USERNAME+","+USER_PASS_HASH+","+
      USER_TIME_CREATED+") "+
    "VALUES (:"+USER_USERNAME+",:"+USER_PASS_HASH+",:"+USER_TIME_CREATED+")",
    &user,
  )
  if err != nil { errman.PrintError(err); return _ErrInsertAcc }

  _, err = tx.Exec(
    "UPDATE "+_FINGERPRINTS+" "+
    "SET "+FINGERPRINT_ACCS_CREATED+" = ?"+
    "WHERE "+FINGERPRINT_FP_ID+" = ?",
    fp.AccsCreated + 1, fp.FpId,
  )
  if err != nil { errman.PrintError(err); return _ErrInsertAcc }

  err = tx.Commit()
  if err != nil { errman.PrintError(err); return _ErrInsertAcc }

  return nil
}

var _UserSortFields = []string{ USER_USERNAME, USER_TIME_CREATED, USER_ACTIVE }

func GetUsers(params SelectParams) ([]User, error) {
  users := []User{}
  err := dbSelectList(&users, _USER_FIELDS, _USERS, params, _UserSortFields)
  if err != nil { return []User{}, _ErrGetAcc }
  return users, nil
}

func GetUserFromUsername(username string) (User, error) {
  user := User{}
  err := dbGet(&user,
    "SELECT * "+
    "FROM "+_USERS+" "+
    "WHERE "+USER_USERNAME+" = ? AND "+USER_ACTIVE+" = true",
    username,
  )
  if err != nil { return User{}, _ErrGetAcc }
  return user, nil
}

func UpdateUserField(username, field string, v any) error {
  _, err := dbUpdateTableField(_USERS, USER_USERNAME, username, field, v)
  if err != nil { errman.PrintError(err); return _ErrModAcc }
  return nil
}