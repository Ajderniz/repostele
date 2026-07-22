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

var _AuthErr = errors.New("Unauthorized")
var _ExpiredErr = errors.New("Session token expired")

func RequireUserAuth() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

      sessionCookie, err := r.Cookie("session_token")
      if err != nil {
        errman.PrintError(err)
        write.ErrorJSON(w, http.StatusUnauthorized, _AuthErr)
        return
      }

      session, err := models.GetSessionFromToken(sessionCookie.Value)
      if err != nil {
        errman.PrintError(err)
        write.ErrorJSON(w, http.StatusUnauthorized, _AuthErr)
        return
      }

      if r.Method != http.MethodGet && r.Method != http.MethodHead {
        csrf := r.Header.Get("X-CSRF-Token")
        if csrf == "" || csrf != session.CSRFToken {
          errman.PrintError(errors.New("'csrf_token' not found"))
          write.ErrorJSON(w, http.StatusUnauthorized, _AuthErr)
          return
        }
      }

      if time.Now().After(time.Unix(session.Expires, 0)) {
        errman.PrintError(_ExpiredErr)
        write.ErrorJSON(w, http.StatusUnauthorized, _ExpiredErr)
        return
      }

      ctx := context.WithValue(r.Context(), models.USER_USERNAME, session.User)
      next.ServeHTTP(w, r.WithContext(ctx))
    })
  }
}

func RequireStaffAuth() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

    })
  }
}