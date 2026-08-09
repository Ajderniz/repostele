package models

import (
  "database/sql"
  "errors"
  "strconv"
  "time"

  "github.com/ajderniz/repostele/pkg/errman"
)

type SessionRole int
const (
  SESSION_ROLE_USER = iota
  SESSION_ROLE_STAFF
)

type Session struct {
  SessionToken string      `db:"session_token"`
  CSRFToken    string      `db:"csrf_token"`
  User         string      `db:"user"          json:"user"`
  Role         SessionRole `db:"role"          json:"role"`
  Starts       int64       `db:"starts"        json:"starts"`
  Expires      int64       `db:"expires"       json:"expires"`
}

const (
  _SESSIONS          = "sessions"
  SESSION_TOKEN      = "session_token"
  SESSION_CSRF_TOKEN = "csrf_token"
  _SESSION_USER      = "user"
  _SESSION_ROLE      = "role"
  _SESSION_STARTS    = "starts"
  _SESSION_EXPIRES   = "expires"
  _SESSION_FIELDS    = SESSION_TOKEN+","+SESSION_CSRF_TOKEN+","+ _SESSION_USER+
                      ","+_SESSION_ROLE+","+_SESSION_STARTS+","+_SESSION_EXPIRES
)

func InsertSession(session Session) error {
  _, err := dbBeginNamedExecAndCommit(
    "INSERT INTO "+_SESSIONS+" ("+_SESSION_FIELDS+") "+
    "VALUES(:"+SESSION_TOKEN+",:"+SESSION_CSRF_TOKEN+",:"+_SESSION_USER+
            ",:"+_SESSION_ROLE+",:"+_SESSION_STARTS+",:"+_SESSION_EXPIRES+")",
    &session,
  )
  if err!=nil{ return errors.New("Could not open session") }
  return nil
}

func GetSessionFromToken(sessionToken string) (Session, error) {
  session := Session{}
  err := dbGet(&session, "SELECT * FROM "+_SESSIONS+" WHERE "+SESSION_TOKEN+" = ?", sessionToken)
  if err != nil {
    errman.PrintError(err)
    if err == sql.ErrNoRows {return Session{},errors.New("Session nonexistent")}
    return Session{}, errors.New("Could not retrieve session information")
  }
  return session, nil
}

func expireTime() int64 {
  return time.Now().Add(-time.Hour).Unix()
}

func CloseSessionForUsername(username string) error {
  _, err := dbUpdateTableField(
    _SESSIONS, _SESSION_EXPIRES, expireTime(), _SESSION_USER, username,
  )
  if err != nil { return errors.New("Could not close session") }
  return nil
}

var (
  _SessionRoleUserStr  = strconv.Itoa(int(SESSION_ROLE_USER))
  _SessionRoleStaffStr = strconv.Itoa(int(SESSION_ROLE_STAFF))
)

func CloseAllSessions(users, staff bool) error {
  query := "UPDATE "+_SESSIONS+" "+
           "SET "+_SESSION_EXPIRES+" = ?"
  if users != staff {
    query += " WHERE "+_SESSION_ROLE+" = "
    if users {
      query += _SessionRoleUserStr
    } else {
      query += _SessionRoleStaffStr
    }
  }
  _, err := dbBeginExecAndCommit(query, expireTime())
  if err != nil { return errors.New("Could not close all sessions") }
  return nil
}

var _SessionSortFields = []string{
  _SESSION_USER, _SESSION_ROLE, _SESSION_STARTS, _SESSION_EXPIRES,
}

func GetActiveSessions(params SelectParams) ([]Session, error) {
  params.Fix(_SessionSortFields)
  sessions := []Session{}
  err := dbSelect(&sessions,
    "SELECT "+_SESSION_USER+","+_SESSION_ROLE+","+_SESSION_STARTS+","+
              _SESSION_EXPIRES+" "+
    "FROM "+_SESSIONS+" "+
    "WHERE ? < "+_SESSION_EXPIRES,
    time.Now().Unix(),
  )
  if dbSelectErr(err) != nil {
    errman.PrintError(err)
    return []Session{}, errors.New("Could not retrieve session list")
  }
  return sessions, nil
}
