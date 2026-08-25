package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
	"github.com/ajderniz/repostele/pkg/pass"
)

var _ErrSessionID = errors.New("'"+SESSION_ID+"' blank")

const (
  _TOKEN_LENGTH = 32
  SESSION_ID    = "session-id"
  _CSRF_TOKEN   = "csrf-token"
)

// TODO: actually check if user is logged in, not just if they have a 'session-id' cookie
func checkSession(r *http.Request) bool {
  _, err := r.Cookie(SESSION_ID)
  return err == nil
}

func openSession(
  w http.ResponseWriter,
  username string,
  role models.SessionRole,
  sessionID, fingerprintID string,
) error {

  sessionToken, err := pass.GenerateToken(_TOKEN_LENGTH)
  csrfToken,    err := pass.GenerateToken(_TOKEN_LENGTH)
  if err != nil { return err }

  expires := time.Now().Add(24 * time.Hour)

  otherSession, err := models.GetLatestOpenSessionFromUsername(username)
  if err != nil { return err }

  if otherSession.SessionToken != "" {
    err = closeSession(w, otherSession.SessionToken)
    if err != nil { return err }
  }
  if sessionID != "" {
    err = closeSession(w, sessionID)
    if err != nil { return err }
  }

  http.SetCookie(w, &http.Cookie{
    Name:     SESSION_ID,
    Value:    sessionToken,
    Expires:  expires,
    HttpOnly: true,
  })
  http.SetCookie(w, &http.Cookie{
    Name:     _CSRF_TOKEN,
    Value:    csrfToken,
    Expires:  expires,
    HttpOnly: false,
  })

  return models.InsertSession(
    models.Session{
      SessionToken: sessionToken,
      CSRFToken:    csrfToken,
      User:         username,
      Role:         role,
      Starts:       time.Now().Unix(),
      Expires:      expires.Unix(),
    },
    fingerprintID,
  )
}

func closeSession(w http.ResponseWriter, sid string) error {
  err := models.CloseSession(sid)
  if err != nil { return err }

  http.SetCookie(w, &http.Cookie{
    Name:     SESSION_ID,
    Value:    "",
    Expires:  time.Now(),
    HttpOnly: true,
  })
  http.SetCookie(w, &http.Cookie{
    Name:     _CSRF_TOKEN,
    Value:    "",
    Expires:  time.Now(),
    HttpOnly: false,
  })

  return nil
}

func CloseSessionForUsername(w http.ResponseWriter, r *http.Request) {
  username, err := bind.URLParam(r, _CREDS_USERNAME, "required")
  if err != nil { serveBadRequestHX(w, err.Error()); return }

  err = models.CloseSessionForUsername(username)
  if err != nil { serveInternalErrHX(w); return }

  serveResponseHX(w, "La sesión fue cerrada", OK, &_NextAction{
    URL: "/",
    Name: "Volver al inicio",
    HTMX: false,
  })
}

func CloseAllSessions(w http.ResponseWriter, r *http.Request) {
  userStr, err := bind.FormValue(r, "users", "boolean")
  staffStr, err := bind.FormValue(r, "staff", "boolean")
  if err != nil { serveBadRequestHX(w, err.Error()); return }

  users, _ := strconv.ParseBool(userStr)
  staff, _ := strconv.ParseBool(staffStr)

  if users || staff {
    err = models.CloseAllSessions(users, staff)
    if err != nil { serveInternalErrHX(w); return }
  }
  serveResponseHX(w, "Se cerraron todas las sesiones deseadas", OK, nil)
}

func Logout(w http.ResponseWriter, r *http.Request) {
  sid, err := r.Cookie(SESSION_ID)
  if err != nil { serveBadRequest(w, r, err); return }
  err = closeSession(w, sid.Value)
  if err != nil { serveBadRequest(w, r, err); return }
  serveMsg(w, r, _MsgLoggedOut)
}

func GetActiveSessions(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { serveBadRequest(w, r, err); return }

  sessions, err := models.GetActiveSessions(params)
  if err != nil { serveInternalErr(w, r); return }

  if len(sessions) <= 0 { serveNoResults(w, r); return }
  serveDataHX(w, r, sessions, "list-sessions")
}
