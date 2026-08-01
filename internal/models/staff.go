package models

import (
	"github.com/ajderniz/repostele/pkg/errman"
)

type Staff struct {
	Username    string `db:"username"`
	PassHash    string `db:"pass_hash"`
	FullName    string `db:"full_name"`
	TimeCreated int64  `db:"time_created"`
	Active      bool   `db:"active"`
	Admin       bool   `db:"admin"`
}

const (
	_STAFF              = "staff"
	STAFF_USERNAME     = "username"
	STAFF_PASS_HASH    = "pass_hash"
	_STAFF_FULL_NAME    = "full_name"
	STAFF_TIME_CREATED = "time_created"
	STAFF_ACTIVE       = "active"
	_STAFF_ADMIN        = "admin"
	//_STAFF_FIELDS       = _STAFF_USERNAME+","+_STAFF_PASS_HASH+","+
	                      //_STAFF_FULL_NAME+","+_STAFF_TIME_CREATED+","+
	                      //_STAFF_ACTIVE+","+_STAFF_ADMIN
)

func InsertStaffAccount(staff Staff) error {
	_, err := dbBeginNamedExecAndCommit(
		"INSERT INTO "+_STAFF+" ("+STAFF_USERNAME+","+STAFF_PASS_HASH+","+
			_STAFF_FULL_NAME+","+STAFF_TIME_CREATED+","+_STAFF_ADMIN+") "+
		"VALUES (:"+STAFF_USERNAME+",:"+STAFF_PASS_HASH+",:"+_STAFF_FULL_NAME+
			",:"+STAFF_TIME_CREATED+",:"+_STAFF_ADMIN+")",
		staff,
	)
	if err != nil { return _ErrInsertAcc }
	return nil
}

var _StaffSortFields = _SortFields{
	STAFF_USERNAME,_STAFF_FULL_NAME,STAFF_TIME_CREATED,STAFF_ACTIVE,_STAFF_ADMIN,
}

func GetStaff(params SelectParams) ([]Staff, error) {
	staff := []Staff{}
	err := dbSelectList(&staff, _STAFF, params, _StaffSortFields)
	if err != nil { return []Staff{},_ErrGetAcc}
	return staff, nil
}

func GetStaffAdmins() ([]Staff, error) {
	admins := []Staff{}
	err := dbSelect(&admins,
		"SELECT * "+
		"FROM "+_STAFF+
		"WHERE "+_STAFF_ADMIN+" = true AND "+STAFF_ACTIVE+" = true",
	)
	if err != nil { return []Staff{}, _ErrGetAccs }
	return admins, nil
}

func GetStaffFromUsername(username string) (Staff, error) {
	staff := Staff{}
	err := dbGet(&staff,
		"SELECT * "+
		"FROM "+_STAFF+" "+
		"WHERE "+STAFF_USERNAME+" = ? AND "+STAFF_ACTIVE+" = true",
		username,
	)
	if err != nil { return Staff{}, _ErrGetAcc }
	return staff, nil
}

func UpdateStaffField(username, field string, v any) error {
	_, err := dbUpdateTableField(_STAFF, STAFF_USERNAME, username, field, v)
  if err != nil { errman.PrintError(err); return _ErrModAcc }
  return nil
}