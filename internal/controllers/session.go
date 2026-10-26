package controllers

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
	"github.com/ajderniz/repostele/pkg/pass"
)

const (
  _TOKEN_LENGTH = 32
  SESSION_ID    = "session-id"
  _CSRF_TOKEN   = "csrf-token"
)

// checkSessionUser is for routes without auth middleware (/htmx/notif);
// elsewhere the middleware already put the session in the request context.
func checkSessionUser(
  r *http.Request,
) (username string, role models.SessionRole, ok bool) {

  sessionCookie, err := r.Cookie(SESSION_ID)
  if err != nil { return "", 0, false }

  session, err := models.GetSessionFromID(sessionCookie.Value)
  if err != nil || session.SessionToken == "" { return "", 0, false }

  if session.Expires <= time.Now().Unix() { return "", 0, false }
  if session.Role != models.APP_ROLE { return "", 0, false }

  return session.User, session.Role, true
}

func openSession(
  w http.ResponseWriter,
  username string,
  role models.SessionRole,
  sessionID, fingerprintID string,
) (models.Session, error) {

  sessionToken, err := pass.GenerateToken(_TOKEN_LENGTH)
  if err != nil { return models.Session{}, err }
  csrfToken, err := pass.GenerateToken(_TOKEN_LENGTH)
  if err != nil { return models.Session{}, err }

  expires := time.Now().Add(24 * time.Hour)

  otherSession, err := models.GetLatestOpenSessionFromUsername(username, role)
  if err != nil { return models.Session{}, err }

  if otherSession.SessionToken != "" {
    err = closeSession(w, otherSession.SessionToken)
    if err != nil { return models.Session{}, err }
  }
  if sessionID != "" {
    err = closeSession(w, sessionID)
    if err != nil { return models.Session{}, err }
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

  session := models.Session{
    SessionToken: sessionToken,
    CSRFToken:    csrfToken,
    User:         username,
    Role:         role,
    Starts:       time.Now().Unix(),
    Expires:      expires.Unix(),
  }
  return session, models.InsertSession(session, fingerprintID)
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
  roleStr, err := bind.URLParam(r, "role", "required,oneof=0 1")
  if err != nil { serveBadRequestHX(w, err.Error()); return }
  role, _ := strconv.Atoi(roleStr)

  err = models.CloseSessionForUsername(username, models.SessionRole(role))
  if err != nil { serveInternalErrHX(w); return }

  serveOOB(w, OK, _HXData{Msg: "La sesión fue cerrada"},
    frag("oob-delete", "session-"+roleStr+"-"+username))
}

func CloseAllSessions(w http.ResponseWriter, r *http.Request) {
  userStr, err := bind.FormValue(r, "users", "boolean")
  if err != nil { serveBadRequestHX(w, err.Error()); return }
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

// Login checks credentials against this build's account table and opens a
// session with this build's role.
func Login(w http.ResponseWriter, r *http.Request) {
  fp, err := checkLoginAttempts(w, r)
  if err != nil { serveErr(w, r, Forbidden, err); return }

  username, password, err := getCredsFromForm(r)
  if err != nil { serveBadRequest(w, r, err); return }

  passHash, err := accountPassHash(username)
  if err != nil { serveInternalErr(w, r); return }
  if passHash == "" { serveMsg(w, r, _MsgAccNotFound); return }

  err = pass.CheckPasswordHash(password, passHash)
  if err != nil { failLogin(w, r, fp); return }

  sessionIDCookie, _ := r.Cookie(SESSION_ID)
  sessionID := ""
  if sessionIDCookie != nil { sessionID = sessionIDCookie.Value }

  session, err := openSession(w, username, models.APP_ROLE, sessionID, fp.Id)
  if err != nil { serveInternalErr(w, r); return }

  // The page is rendered in this request, so its nav must see the new session.
  serveMsg(w, r.WithContext(models.WithSession(r.Context(), session)), _MsgLoggedIn)
}

func Logout(w http.ResponseWriter, r *http.Request) {
  sid, err := r.Cookie(SESSION_ID)
  if err != nil { serveBadRequest(w, r, err); return }
  err = closeSession(w, sid.Value)
  if err != nil { serveBadRequest(w, r, err); return }

  if isHX(r) {
    w.Header().Set("HX-Redirect", "/login?msg="+url.QueryEscape("Sesión cerrada"))
    w.WriteHeader(OK)
    return
  }
}

func GetActiveSessions(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { serveBadRequest(w, r, err); return }

  sessions, err := models.GetActiveSessions(&params)
  if err != nil { serveInternalErr(w, r); return }

  serveDataHX(w, r, map[string]any{
    "Sessions": sessions,
    "Params":   params,
    "HasNext":  len(sessions) == params.Limit,
  }, "list-sessions")
}
