package controllers

import (
	"errors"
	"net/http"

	"github.com/ajderniz/repostele/internal/models"
)

const (
  _MAX_REG_ACCS = 1
  _MAX_FAILED_LOGINS = 3
)

var (
	_ErrAlreadyInit  = errors.New("Already initialized")
	_ErrSameUsername = errors.New("Username already exists")
	_ErrSamePassword = errors.New("Password is the same")
  _ErrGetAcc = errors.New("Could not retrieve account information")

	_MsgAccNotFound = "Account not found"
  _MsgAccCreated = "Account registered successfully"
	_MsgLoggedOut = "Logged out successfully"
	_MsgAccDeactivated = "Account deactivated successfully"
	_MsgUsernameChanged = "Username changed successfully"
	_MsgPasswordChanged = "Password changed successfully"
)

func checkLoginAttempts(w http.ResponseWriter, r *http.Request) (fp models.Fingerprint, err error) {
  fp = r.Context().Value(models.FINGERPRINT).(models.Fingerprint)
  if _MAX_FAILED_LOGINS <= fp.FailedLogins {
    err = errors.New("Max login attempts reached")
  }
  return 
}

func failLogin(w http.ResponseWriter, fp models.Fingerprint) {
	// already a fail, so don't check for errors
  models.UpdateFingerprintField(
    fp.Id, models.FINGERPRINT_FAILED_LOGINS, fp.FailedLogins + 1,
  )
  w.WriteHeader(http.StatusUnauthorized)
}
	