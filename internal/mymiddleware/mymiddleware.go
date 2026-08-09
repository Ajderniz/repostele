package mymiddleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ajderniz/repostele/internal/controllers"
	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/errman"
	"github.com/ajderniz/repostele/pkg/write"
	"github.com/anhnmt/go-fingerprint"
)

var (
  _ErrAuth = errors.New("Unauthorized")
  _ErrGetSession = errors.New("Could not retrieve session info")
  _ErrExpired = errors.New("Session expired")
)

func getSession(w http.ResponseWriter, r *http.Request) (models.Session, int, error){
  sessionCookie, err := r.Cookie(controllers.SESSION_ID)
  if err != nil {
    errman.PrintError(err)
    return models.Session{}, http.StatusUnauthorized, _ErrAuth
  }

  session, err := models.GetSessionFromToken(sessionCookie.Value)
  if err != nil {
    errman.PrintError(err)
    return models.Session{}, http.StatusInternalServerError, _ErrGetSession
  }
  if session.SessionToken == "" {
    errman.PrintError(_ErrGetSession)
    return models.Session{}, http.StatusInternalServerError, _ErrGetSession 
  }

  if r.Method != http.MethodGet && r.Method != http.MethodHead {
    csrf := r.Header.Get("X-CSRF-Token")
    if csrf == "" {
      errman.PrintError(errors.New("'csrf-token' not found"))
      return models.Session{}, http.StatusUnauthorized, _ErrAuth
    }
    if csrf != session.CSRFToken {
      errman.PrintError(errors.New("Invalid CSRF token"))
      return models.Session{}, http.StatusUnauthorized, _ErrAuth
    }
  }

  if session.Expires <= time.Now().Unix() {
    errman.PrintError(_ErrExpired)
    return models.Session{}, http.StatusUnauthorized, _ErrExpired
  }

  return session, http.StatusOK, nil
}

func RequireUserAuth() func(next http.Handler) http.Handler {
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

func RequireStaffAuth() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      session, status, err := getSession(w, r)
      if err != nil { 
        if status == http.StatusUnauthorized { w.WriteHeader(status); return }
        write.Error(w, status, err); return
      }
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
      if !staff.Admin { w.WriteHeader(http.StatusUnauthorized); return }
      next.ServeHTTP(w, r)
    })
  }
}

func CheckInit() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      admins, err := models.GetStaffAdmins()
      if err != nil || len(admins) <= 0 {
        errman.PrintError(errors.New("System not initialized"))
        w.WriteHeader(http.StatusForbidden)
        return
      }
      next.ServeHTTP(w, r)
    })
  }
}

func GetFingerprint() func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

      id := fingerprint.NewFingerprint(r).ID

      fp, err := models.GetFingerPrintFromID(id)
      if err != nil {
        errman.PrintError(err)
        w.WriteHeader(http.StatusInternalServerError)
        return
      }

      if fp.FpId == "" {
        fp.FpId           = id
        fp.Expires      = time.Now().Add(time.Hour).Unix()
        fp.FailedLogins = 0
        fp.AccsCreated  = 0
        err = models.InsertFingerprint(fp)
        if err != nil {
          errman.PrintError(err)
          w.WriteHeader(http.StatusInternalServerError)
          return
        }
      }

      ctx := context.WithValue(r.Context(), models.FINGERPRINT, fp)
      next.ServeHTTP(w, r.WithContext(ctx))
    })
  }
}
