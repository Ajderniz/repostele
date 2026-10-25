//go:build USER

package mymiddleware

import (
	"log/slog"
	"net/http"

	"github.com/ajderniz/repostele/internal/models"
)

// Only customer sessions are valid on this app.
const _APP_ROLE = models.SESSION_ROLE_USER

func CheckInit() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      if !models.CheckInit() {
        slog.Error("El sistema no ha sido configurado")
        w.WriteHeader(http.StatusForbidden)
        return
      }
      next.ServeHTTP(w, r)
    })
  }
}
