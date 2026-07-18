package models

import (
  "database/sql"
  "errors"
  "time"

  "github.com/ajderniz/repostele/pkg/errman"
)

type Session struct {
  SessionToken string `db:"session_token"`
  CSRFToken    string `db:"csrf_token"`
  User         string `db:"user"`
  Starts       int64  `db:"starts"`
  Expires      int64  `db:"expires"`
}

const (
  _SESSIONS          = "sessions"
  SESSION_TOKEN      = "session_token"
  SESSION_CSRF_TOKEN = "csrf_token"
  _SESSION_USER      = "user"
  _SESSION_STARTS    = "starts"
  _SESSION_EXPIRES   = "expires"
  _SESSION_FIELDS    = SESSION_TOKEN+","+SESSION_CSRF_TOKEN+","+ _SESSION_USER+
                       ","+_SESSION_STARTS+","+_SESSION_EXPIRES
)

var _OpenSessionErr = errors.New("Could not open session")

func OpenSession(session Session) error {
  tx, err := _DB.Beginx()
  if err != nil { errman.PrintError(err); return _OpenSessionErr }

  _, err = tx.NamedExec(
    "INSERT INTO "+_SESSIONS+" ("+_SESSION_FIELDS+") "+
    "VALUES(:"+SESSION_TOKEN+",:"+SESSION_CSRF_TOKEN+",:"+_SESSION_USER+
            ",:"+_SESSION_STARTS+",:"+_SESSION_EXPIRES+")",
    &session,
  )
  if err != nil { errman.PrintError(err); return _OpenSessionErr }

  err = tx.Commit()
  if err != nil { errman.PrintError(err); return _OpenSessionErr }

  return nil
}

func GetSessionFromToken(sessionToken string) (Session, error) {
  session := Session{}
  err := _DB.Get(&session,
    "SELECT * "+
    "FROM "+_SESSIONS+" "+
    "WHERE "+SESSION_TOKEN+" = $1",
    sessionToken,
  )
  if err != nil {
    errman.PrintError(err)
    if err == sql.ErrNoRows { return Session{}, errors.New("Session nonexistent") }
    return Session{}, errors.New("Could not retreive session information")
  }
  return session, nil
}

var _CloseSessionErr = errors.New("Could not close session")

func CloseSession(sessionToken string) error {
  tx, err := _DB.Beginx()
  if err != nil { errman.PrintError(err); return _CloseSessionErr }

  _, err = tx.Exec(
    "UPDATE "+_SESSIONS+" "+
    "SET "+_SESSION_EXPIRES+" = $1 "+
    "WHERE "+SESSION_TOKEN+" = $2",
    time.Now().Add(-time.Hour).Unix(), sessionToken,
  )
  if err != nil { errman.PrintError(err); return _CloseSessionErr }

  err = tx.Commit()
  if err != nil { errman.PrintError(err); return _CloseSessionErr }
  
  return nil
}