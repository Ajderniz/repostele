//go:build STAFF

package mymiddleware

import (
	"context"
	"net/http"

	"github.com/ajderniz/repostele/internal/models"
)

func RequireAuth() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      session, status, err := getSession(w, r)
      if err != nil {
        if status == http.StatusUnauthorized { w.WriteHeader(status); return }
        w.WriteHeader(http.StatusInternalServerError)
      }
      ctx := context.WithValue(r.Context(), models.STAFF_USERNAME, session.User)
      next.ServeHTTP(w, r.WithContext(ctx))
    })
  }
}

func CheckInit() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      if !models.CheckInit(){
        http.Redirect(w, r, "/init", http.StatusPermanentRedirect)
        return
      }
      next.ServeHTTP(w, r)
    })
  }
}
