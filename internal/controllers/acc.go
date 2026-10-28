package controllers

import (
	"errors"
	"net/http"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
	"github.com/ajderniz/repostele/pkg/pass"
)

const (
  _MAX_REG_ACCS = 1
  _MAX_FAILED_LOGINS = 3
)

var (
	_ErrBadCreds = errors.New("Credenciales inválidos")
	_ErrSamePassword = errors.New("La contraseña es idéntica a la anterior")

	_MsgAccNotFound = "Cuenta no encontrada"
  _MsgAccCreated = "Registro exitoso"
	_MsgLoggedIn = "Inicio de sesión exitoso"
	_MsgAccDeactivated = "La cuenta fue desactivada"
	_MsgPasswordChanged = "Se cambió la contraseña"
)

func checkLoginAttempts(w http.ResponseWriter, r *http.Request) (fp models.Fingerprint, err error) {
  fp = models.FingerprintFromContext(r.Context())
  if _MAX_FAILED_LOGINS <= fp.FailedLogins {
    err = errors.New("Se alcanzó el límite de intentos de inicio de sesión")
  }
  return 
}

func failLogin(w http.ResponseWriter, r *http.Request, fp models.Fingerprint) {
	// already a fail, so don't check for errors
  models.UpdateFingerprintField(
    fp.Id, models.FINGERPRINT_FAILED_LOGINS, fp.FailedLogins + 1,
  )
  serveResponse(w, r, nil, Unauthorized, _ErrBadCreds)
}

// Each build defines accountPassHash / setAccountPassHash for its own table
// (users or staff) and selfDeactivate for its own deactivation rules.
// accountPassHash returns "" when the account doesn't exist or is inactive.

func updatePassword(username, oldPassword, newPassword string) (int, error) {
  if oldPassword == newPassword { return BadRequest, _ErrSamePassword }

  passHash, err := accountPassHash(username)
  if err != nil { return InternalServerError, err }
  if passHash == "" { return NotFound, errors.New(_MsgAccNotFound) }

  err = pass.CheckPasswordHash(oldPassword, passHash)
  if err != nil { return Unauthorized, err }

  newHash, err := pass.HashPassword(newPassword)
  if err != nil { return InternalServerError, err }

  err = setAccountPassHash(username, newHash)
  if err != nil { return InternalServerError, err }

  return OK, nil
}

func SelfUpdatePassword(w http.ResponseWriter, r *http.Request) {
  oldPassword, newPassword, err := getNewPasswordFromForm(r)
  if err != nil { serveBadRequestHX(w, err.Error()); return }

  status, err := updatePassword(ctxUsername(r), oldPassword, newPassword)
  if err != nil { serveStatusErrHX(w, r, status, err); return }

  serveResponseHX(w, _MsgPasswordChanged, OK, nil)
}

func SelfDeactivateAccount(w http.ResponseWriter, r *http.Request) {
  username := ctxUsername(r)
  password, err := bind.FormValue(r, _CREDS_PASSWORD, _CREDS_VALIDATE)
  if err != nil { serveInternalErrHX(w, r, err); return }

  passHash, err := accountPassHash(username)
  if err != nil || passHash == "" { serveInternalErrHX(w, r, err); return }

  err = pass.CheckPasswordHash(password, passHash)
  if err != nil {
    serveResponseHX(w, _ErrBadCreds.Error(), Unauthorized, nil)
    return
  }

  status, err := selfDeactivate(w, r, username)
  if err != nil { serveStatusErrHX(w, r, status, err); return }

  if isHX(r) {
    w.Header().Set("HX-Redirect", "/login")
    w.WriteHeader(OK)
    return
  }
  serveMsg(w, r, _MsgAccDeactivated)
}

// closeAccountSessions ends the caller's own session (self) or every session
// the account has for role.
func closeAccountSessions(
  w http.ResponseWriter,
  r *http.Request,
  username string,
  role models.SessionRole,
  self bool,
) (int, error) {
  if self {
    sid, err := r.Cookie(SESSION_ID)
    if err != nil { return BadRequest, err }
    if err := closeSession(w, sid.Value); err != nil { return InternalServerError, err }
    return OK, nil
  }
  if err := models.CloseSessionForUsername(username, role); err != nil {
    return InternalServerError, err
  }
  return OK, nil
}
