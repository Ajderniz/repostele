//go:build USER

package mymiddleware

import (
	"errors"
	"net/http"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/internal/reqlog"
)

func CheckInit() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      if !models.CheckInit() {
        reqlog.Error(r, errors.New("El sistema no ha sido configurado"))
        w.WriteHeader(http.StatusForbidden)
        return
      }
      next.ServeHTTP(w, r)
    })
  }
}
