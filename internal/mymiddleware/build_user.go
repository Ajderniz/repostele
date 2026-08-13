//go:build USER

package mymiddleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/write"
)

func RequireAuth() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      session, status, err := getSession(w, r)
      if err != nil {
        if status == http.StatusUnauthorized { w.WriteHeader(status); return }
        write.Error(w, status, err); return
      }
      ctx := context.WithValue(r.Context(), models.USER_USERNAME, session.User)
      next.ServeHTTP(w, r.WithContext(ctx))
    })
  }
}

func CheckInit() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      if !models.CheckInit() {
        slog.Error("System not initialized")
        w.WriteHeader(http.StatusForbidden)
        return
      }
      next.ServeHTTP(w, r)
    })
  }
}
