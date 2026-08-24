package mymiddleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ajderniz/repostele/internal/controllers"
	"github.com/ajderniz/repostele/internal/models"
	"github.com/anhnmt/go-fingerprint"
)

const (
  _CREDS_USERNAME = "username"
)

var (
  _ErrAuth = errors.New("Sin autorización")
  _ErrGetSession = errors.New("No se pudo acceder a la información de sesión")
  _ErrExpired = errors.New("La sesión expiró")
)

func getSession(w http.ResponseWriter, r *http.Request) (models.Session, int, error){
  sessionCookie, err := r.Cookie(controllers.SESSION_ID)
  if err != nil {
    slog.Error(err.Error())
    return models.Session{}, http.StatusUnauthorized, _ErrAuth
  }

  session, err := models.GetSessionFromID(sessionCookie.Value)
  if err != nil {
    slog.Error(err.Error())
    return models.Session{}, http.StatusInternalServerError, _ErrGetSession
  }
  if session.SessionToken == "" {
    slog.Error(_ErrGetSession.Error())
    return models.Session{}, http.StatusNotFound, _ErrGetSession
  }

  if r.Method != http.MethodGet && r.Method != http.MethodHead {
    csrf := r.Header.Get("X-CSRF-Token")
    if csrf == "" {
      slog.Error("No se encontró la cookie llamada 'csrf-token'")
      return models.Session{}, http.StatusUnauthorized, _ErrAuth
    }
    if csrf != session.CSRFToken {
      slog.Error("Token CSRF inválido")
      return models.Session{}, http.StatusUnauthorized, _ErrAuth
    }
  }

  if session.Expires <= time.Now().Unix() {
    slog.Error(_ErrExpired.Error())
    return models.Session{}, http.StatusUnauthorized, _ErrExpired
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
        ctx := context.WithValue(r.Context(), _CREDS_USERNAME, session.User)
        next.ServeHTTP(w, r.WithContext(ctx))
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
        if status == http.StatusUnauthorized { w.WriteHeader(status); return }
        w.WriteHeader(http.StatusInternalServerError)
      }
      ctx := context.WithValue(r.Context(), _CREDS_USERNAME, session.User)
      next.ServeHTTP(w, r.WithContext(ctx))
    })
  }
}

func GetFingerprint() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

      id := fingerprint.NewFingerprint(r).ID

      fp, err := models.GetFingerPrintFromID(id)
      if err != nil {
        slog.Error(err.Error())
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
          slog.Error(err.Error())
          w.WriteHeader(http.StatusInternalServerError)
          return
        }
      }

      ctx := context.WithValue(r.Context(), models.FINGERPRINT, fp)
      next.ServeHTTP(w, r.WithContext(ctx))
    })
  }
}
