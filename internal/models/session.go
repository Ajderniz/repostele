package models

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"
)

var ErrSessionNotFound = errors.New("La sesión no existe")

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

// Request context keys. A private type keeps them from colliding with other packages.
type _CtxKey int
const (
  _CTX_SESSION _CtxKey = iota
  _CTX_FINGERPRINT
)

func WithSession(ctx context.Context, s Session) context.Context {
  return context.WithValue(ctx, _CTX_SESSION, s)
}

// SessionFromContext returns the session set by the auth middleware, if any.
func SessionFromContext(ctx context.Context) (Session, bool) {
  s, ok := ctx.Value(_CTX_SESSION).(Session)
  return s, ok
}

func WithFingerprint(ctx context.Context, fp Fingerprint) context.Context {
  return context.WithValue(ctx, _CTX_FINGERPRINT, fp)
}

func FingerprintFromContext(ctx context.Context) Fingerprint {
  fp, _ := ctx.Value(_CTX_FINGERPRINT).(Fingerprint)
  return fp
}

var _ErrOpenSession = errors.New("No se pudo abrir una nueva sesión")

func InsertSession(session Session, fingerprintID string) error {
  tx, err := _DB.Beginx()
  if err != nil { slog.Error(err.Error()); return _ErrOpenSession }

  _, err = tx.NamedExec(
    "INSERT INTO "+_SESSIONS+" ("+_SESSION_FIELDS+") "+
    "VALUES(:"+SESSION_TOKEN+",:"+SESSION_CSRF_TOKEN+",:"+_SESSION_USER+
            ",:"+_SESSION_ROLE+",:"+_SESSION_STARTS+",:"+_SESSION_EXPIRES+")",
    &session,
  )
  if err != nil { tx.Rollback(); slog.Error(err.Error()); return _ErrOpenSession}

  _, err = tx.Exec(
    "UPDATE "+_FINGERPRINTS+" SET "+_FINGERPRINT_USER+" = ? "+
    "WHERE "+FINGERPRINT_ID+" = ?",
    session.User, fingerprintID,
  )
  if err != nil { tx.Rollback(); slog.Error(err.Error()); return _ErrOpenSession}

  err = tx.Commit()
  if err != nil { tx.Rollback(); slog.Error(err.Error()); return _ErrOpenSession}

  return nil
}

func GetSessionFromID(sid string) (Session, error) {
  session := Session{}
  err := dbGet(&session,
    "SELECT * FROM "+_SESSIONS+" WHERE "+SESSION_TOKEN+" = ?",
    sid,
  )
  if err != nil {
    slog.Error(err.Error())
    return Session{}, errors.New(
      "No se pudo acceder a la información de la sesión",
    )
  }
  // dbGet maps sql.ErrNoRows to nil, so a missing row shows up as an empty token.
  if session.SessionToken == "" {
    return Session{}, ErrSessionNotFound
  }
  return session, nil
}

// Usernames are unique per table, not across users and staff, so session
// lookups by username also filter by role.
func GetLatestOpenSessionFromUsername(username string, role SessionRole) (Session, error) {
  session := Session{}
  err := dbGet(&session,
    "SELECT * "+
    "FROM "+_SESSIONS+" "+
    "WHERE "+_SESSION_USER+" = ? AND "+_SESSION_ROLE+" = ? AND ? < "+_SESSION_EXPIRES+" "+
    "ORDER BY "+_SESSION_EXPIRES+" DESC "+
    "LIMIT 1",
    username, role, time.Now().Unix(),
  )
  if dbSelectErr(err) != nil { return Session{}, err }
  return session, nil
}

func CloseSession(sessionID string) error {
  _, err := dbUpdateTableField(
    _SESSIONS, _SESSION_EXPIRES, time.Now().Unix(), SESSION_TOKEN, sessionID,
  )
  if err != nil { return _ErrCloseSession }
  return nil
}

func CloseSessionForUsername(username string, role SessionRole) error {
  _, err := dbBeginExecAndCommit(
    "UPDATE "+_SESSIONS+" SET "+_SESSION_EXPIRES+" = ? "+
    "WHERE "+_SESSION_USER+" = ? AND "+_SESSION_ROLE+" = ?",
    time.Now().Unix(), username, role,
  )
  if err != nil { return _ErrCloseSession }
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
  _, err := dbBeginExecAndCommit(query, time.Now().Unix())
  if err != nil { return errors.New("No se pudo cerrar todas las sesiones") }
  return nil
}

var _SessionSortFields = []string{
  _SESSION_USER, _SESSION_ROLE, _SESSION_STARTS, _SESSION_EXPIRES,
}

func GetActiveSessions(params *SelectParams) ([]Session, error) {
  params.Fix(_SessionSortFields)
  sessions := []Session{}
  err := dbSelect(&sessions,
    "SELECT "+_SESSION_USER+","+_SESSION_ROLE+","+_SESSION_STARTS+","+
              _SESSION_EXPIRES+" "+
    "FROM "+_SESSIONS+" "+
    "WHERE ? < "+_SESSION_EXPIRES+" "+
    "ORDER BY "+params.Sort+" "+string(params.Dir)+" "+
    "LIMIT ?, ?",
    time.Now().Unix(), params.Start, params.Limit,
  )
  if dbSelectErr(err) != nil {
    slog.Error(err.Error())
    return []Session{}, errors.New("No se pudo acceder a la lista de sesiones")
  }
  return sessions, nil
}
