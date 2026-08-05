package mymiddleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/errman"
	"github.com/ajderniz/repostele/pkg/write"
)

var _ErrAuth = errors.New("Unauthorized")
var _ErrExpired = errors.New("Session token expired")

func getSession(w http.ResponseWriter, r *http.Request) (models.Session,error) {
  sessionCookie, err := r.Cookie("session_token")
  if err != nil { errman.PrintError(err); return models.Session{}, _ErrAuth }

  session, err := models.GetSessionFromToken(sessionCookie.Value)
  if err != nil { errman.PrintError(err); return models.Session{}, _ErrAuth }

  if r.Method != http.MethodGet && r.Method != http.MethodHead {
    csrf := r.Header.Get("X-CSRF-Token")
    if csrf == "" || csrf != session.CSRFToken {
      errman.PrintError(errors.New("'csrf_token' not found"))
      return models.Session{}, _ErrAuth
    }
  }

  if time.Now().After(time.Unix(session.Expires, 0)) {
    errman.PrintError(_ErrExpired)
    return models.Session{}, _ErrExpired
  }

  return session, nil
}

func RequireUserAuth() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      session, err := getSession(w, r)
      if err != nil { write.Error(w, http.StatusUnauthorized, err); return }
      ctx := context.WithValue(r.Context(), models.USER_USERNAME, session.User)
      next.ServeHTTP(w, r.WithContext(ctx))
    })
  }
}

func RequireStaffAuth() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      session, err := getSession(w, r)
      if err != nil { write.Error(w, http.StatusUnauthorized, err); return }
      ctx := context.WithValue(r.Context(), models.STAFF_USERNAME, session.User)
      next.ServeHTTP(w, r.WithContext(ctx))
    })
  }
}

// Called AFTER staff authentication for the '/staff' path
func RequireAdminAuth() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      username := r.Context().Value(models.STAFF_USERNAME).(string)
      staff, err := models.GetStaffFromUsername(username)
      if err != nil { 
        write.Error(w, http.StatusInternalServerError, err)
        return
      }
      if staff.Username == "" {
        write.Error(w, http.StatusBadRequest, errors.New("Bad username"))
        return
      }
      if !staff.Admin {
        write.Error(w, http.StatusUnauthorized, _ErrAuth)
        return
      }
      next.ServeHTTP(w, r)
    })
  }
}
