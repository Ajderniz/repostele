package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
	"github.com/ajderniz/repostele/pkg/errman"
	"github.com/ajderniz/repostele/pkg/pass"
	"github.com/ajderniz/repostele/pkg/write"
)

var _ErrSessionToken = errors.New("'session_token' cookie not found")

const (
  _TOKEN_LENGTH = 32
  SESSION_ID    = "session-id"
  _CSRF_TOKEN   = "csrf-token"
)

func openSession(w http.ResponseWriter, username string, role models.SessionRole) error {
  sessionToken, err := pass.GenerateToken(_TOKEN_LENGTH)
  csrfToken,    err := pass.GenerateToken(_TOKEN_LENGTH)
  if err != nil { return err }

  expires := time.Now().Add(24 * time.Hour)

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

  return models.InsertSession(models.Session{
    SessionToken: sessionToken,
    CSRFToken:    csrfToken,
    User:         username,
    Role:         role,
    Starts:       time.Now().Unix(),
    Expires:      expires.Unix(),
  })
}

func closeSession(w http.ResponseWriter, r *http.Request) error {
  sessionID, err := r.Cookie(SESSION_ID)
  if err != nil {
    errman.PrintError(err)
    return _ErrSessionToken
  }
  if sessionID.Value == "" {
    errman.PrintError(errors.New("session_token blank"))
    return _ErrSessionToken
  }

  err = models.CloseSessionForUsername(sessionID.Value)
  if err != nil { return err }

  http.SetCookie(w, &http.Cookie{
    Name:     SESSION_ID,
    Value:    "",
    Expires:  time.Now().Add(-time.Hour),
    HttpOnly: true,
  })
  http.SetCookie(w, &http.Cookie{
    Name:     _CSRF_TOKEN,
    Value:    "",
    Expires:  time.Now().Add(-time.Hour),
    HttpOnly: false,
  })

  return nil
}

func CloseSessionForUsername(w http.ResponseWriter, r *http.Request) {
  sessionID, err := bind.URLParam(r, _CREDS_USERNAME, "required")
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  err = models.CloseSessionForUsername(sessionID)
  if err != nil { write.Error(w, http.StatusInternalServerError, err); return }

  write.Msg(w, "Session closed successfully")
}

func CloseAllSessions(w http.ResponseWriter, r *http.Request) {
  userStr, err := bind.FormValue(r, "users", "boolean")
  staffStr, err  := bind.FormValue(r, "staff", "boolean")
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }
  users, _ := strconv.ParseBool(userStr)
  staff, _ := strconv.ParseBool(staffStr)

  err = models.CloseAllSessions(users, staff)
  if err != nil { write.Error(w, http.StatusInternalServerError, err); return }

  write.Msg(w, "All requested sessions closed successfully")
}

func Logout(w http.ResponseWriter, r *http.Request) {
  err := closeSession(w, r)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }
  write.Msg(w, _MsgLoggedOut)
}

func GetActiveSessions(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  sessions, err := models.GetActiveSessions(params)
  if err != nil { write.Error(w, http.StatusInternalServerError, err); return }

  if len(sessions) <= 0 { write.Data(w, _DataNoResults); return }
  write.Data(w, sessions)
}
