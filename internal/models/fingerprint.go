package models

import (
	"errors"
	"time"
)

type Fingerprint struct {
	Id           int    `db:"id"`
	FpId         string `db:"fp_id"`
	Expires      int64  `db:"expires"`
	FailedLogins int    `db:"failed_logins"`
	AccsCreated  int    `db:"accs_created"`
}

const (
	FINGERPRINT = "fingerprint"
	_FINGERPRINTS = "fingerprints"
	//_FINGERPRINTS_ID = "id"
	FINGERPRINT_FP_ID = "fp_id"
	_FINGERPRINT_EXPIRES = "expires"
	FINGERPRINT_FAILED_LOGINS = "failed_logins"
	FINGERPRINT_ACCS_CREATED = "accs_created"
	_FINGERPRINT_FIELDS = /*_FINGERPRINTS_ID+","+*/FINGERPRINT_FP_ID+","+
												_FINGERPRINT_EXPIRES+","+FINGERPRINT_FAILED_LOGINS+","+
												FINGERPRINT_ACCS_CREATED
)

func InsertFingerprint(fg Fingerprint) (error) {
	_, err := dbBeginNamedExecAndCommit(
		"INSERT INTO "+_FINGERPRINTS+" ("+_FINGERPRINT_FIELDS+") "+
		"VALUES (:"/*+_FINGERPRINTS_ID+",:"*/+FINGERPRINT_FP_ID+",:"+
			_FINGERPRINT_EXPIRES+",:"+FINGERPRINT_FAILED_LOGINS+",:"+
			FINGERPRINT_ACCS_CREATED+")",
		&fg,
	)
	if err != nil { return errors.New("Could not save fingerprint") }
	return nil
}

func GetFingerPrintFromID(id string) (Fingerprint, error) {
	fg := Fingerprint{}
	err := dbGet(
		&fg,
		"SELECT * FROM "+_FINGERPRINTS+" "+
		"WHERE "+FINGERPRINT_FP_ID+" = ? AND ? < "+_FINGERPRINT_EXPIRES,
		id, time.Now().Unix(),
	)
	if err != nil {
		return Fingerprint{}, errors.New("Could not retrieve fingerprint")
	}
	return fg, nil
}

func UpdateFingerprintField(id, field string, v any) error {
	_, err := dbUpdateTableField(_FINGERPRINTS, field, v, FINGERPRINT_FP_ID, id)
	if err != nil { return errors.New("Could not update fingerprint") }
	return nil
}