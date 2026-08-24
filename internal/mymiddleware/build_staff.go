//go:build STAFF

package mymiddleware

import (
	"net/http"

	"github.com/ajderniz/repostele/internal/models"
)

// Called AFTER staff authentication for the '/staff' path
func RequireAdminAuth() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      username := r.Context().Value(models.STAFF_USERNAME).(string)
      staff, err := models.GetStaffFromUsername(username)
      if err != nil { w.WriteHeader(http.StatusInternalServerError); return }
      if staff.Username == "" { w.WriteHeader(http.StatusBadRequest); return }
      if !staff.Admin { w.WriteHeader(http.StatusUnauthorized); return }
      next.ServeHTTP(w, r)
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
