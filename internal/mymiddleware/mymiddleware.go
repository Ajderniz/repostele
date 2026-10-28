package mymiddleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/ajderniz/repostele/internal/controllers"
	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/internal/reqlog"
	"github.com/anhnmt/go-fingerprint"
)


var (
  _ErrAuth = errors.New("Sin autorización")
  _ErrGetSession = errors.New("No se pudo acceder a la información de sesión")
  _ErrExpired = errors.New("La sesión expiró")
)

func getSession(w http.ResponseWriter, r *http.Request) (models.Session, int, error){
  sessionCookie, err := r.Cookie(controllers.SESSION_ID)
  if err != nil {
    return models.Session{}, http.StatusUnauthorized, _ErrAuth
  }

  session, err := models.GetSessionFromID(sessionCookie.Value)
  if errors.Is(err, models.ErrSessionNotFound) {
    return models.Session{}, http.StatusNotFound, _ErrGetSession
  }
  if err != nil {
    reqlog.Error(r, err)
    return models.Session{}, http.StatusInternalServerError, _ErrGetSession
  }

  if r.Method != http.MethodGet && r.Method != http.MethodHead {
    csrf := r.Header.Get("X-CSRF-Token")
    if csrf == "" {
      reqlog.Error(r, errors.New("No se encontró el header X-CSRF-Token"))
      return models.Session{}, http.StatusUnauthorized, _ErrAuth
    }
    if csrf != session.CSRFToken {
      reqlog.Error(r, errors.New("Token CSRF inválido"))
      return models.Session{}, http.StatusUnauthorized, _ErrAuth
    }
  }

  if session.Expires <= time.Now().Unix() {
    reqlog.Error(r, _ErrExpired)
    return models.Session{}, http.StatusUnauthorized, _ErrExpired
  }

  // Both apps share the DB and cookies are not port-scoped, so a session
  // from the other app must not pass here.
  if session.Role != models.APP_ROLE {
    return models.Session{}, http.StatusUnauthorized, _ErrAuth
  }

  return session, http.StatusOK, nil
}

func OptUsername() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      session, status, err := getSession(w, r)
      if err != nil && status == http.StatusInternalServerError {
        w.WriteHeader(status)
        return
      }
      if session.User != "" {
        reqlog.SetUser(r.Context(), session.User)
        next.ServeHTTP(w, r.WithContext(models.WithSession(r.Context(), session)))
        return
      }
      next.ServeHTTP(w, r)
    })
  }
}

func RequireAuth() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      session, status, err := getSession(w, r)
      if err != nil {
        // Unknown or expired sessions (401/404) mean "log in again"; only real failures are 500.
        if status == http.StatusInternalServerError { w.WriteHeader(status); return }
        w.WriteHeader(http.StatusUnauthorized)
        return
      }
      reqlog.SetUser(r.Context(), session.User)
      next.ServeHTTP(w, r.WithContext(models.WithSession(r.Context(), session)))
    })
  }
}

func GetFingerprint() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

      id := fingerprint.NewFingerprint(r).ID

      fp, err := models.GetFingerPrintFromID(id)
      if err != nil {
        reqlog.Error(r, err)
        w.WriteHeader(http.StatusInternalServerError)
        return
      }

      if fp.Id == "" {
        fp.Id           = id
        fp.Expires      = time.Now().Add(time.Hour).Unix()
        fp.FailedLogins = 0
        fp.AccsCreated  = 0
        err = models.InsertFingerprint(fp)
        if err != nil {
          reqlog.Error(r, err)
          w.WriteHeader(http.StatusInternalServerError)
          return
        }
      }

      next.ServeHTTP(w, r.WithContext(models.WithFingerprint(r.Context(), fp)))
    })
  }
}
