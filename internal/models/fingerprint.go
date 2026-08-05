package models

import "errors"

type Fingerprint struct {
	Id      string `db:"id"`
	Expires int64  `db:"expires"`
}

const (
	_FINGERPRINTS = "fingerprints"
	FINGERPRINT_ID = "id"
	_FINGERPRINT_EXPIRES = "expires"
	_FINGERPRINT_FIELDS = FINGERPRINT_ID+","+_FINGERPRINT_EXPIRES
)

func InsertFingerprint(fg Fingerprint) (error) {
	_, err := dbBeginNamedExecAndCommit(
		"INSERT INTO "+_FINGERPRINTS+" ("+_FINGERPRINT_FIELDS+") "+
		"VALUES (:"+FINGERPRINT_ID+",:"+_FINGERPRINT_EXPIRES+")",
		&fg,
	)
	if err != nil { return errors.New("Could not save fingerprint") }
	return nil
}

func GetFingerPrintFromID(id string) (Fingerprint, error) {
	fg := Fingerprint{}
	err := dbGetRecord(&fg, _FINGERPRINTS, FINGERPRINT_ID, id)
	if err != nil {
		return Fingerprint{}, errors.New("Could not retrieve fingerprint")
	}
	return fg, nil
}