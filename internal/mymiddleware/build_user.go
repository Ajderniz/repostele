//go:build USER

package mymiddleware

import (
	"log/slog"
	"net/http"

	"github.com/ajderniz/repostele/internal/models"
)

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
