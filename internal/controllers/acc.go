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
	_ErrBadCreds = errors.New("Credenciales inválidos")
	_ErrAlreadyInit  = errors.New("El sistema ya está configurado")
	_ErrSameUsername = errors.New("El usuario ya existe")
	_ErrSamePassword = errors.New("La contraseña es idéntica a la anterior")
  _ErrGetAcc = errors.New("No se pudo acceder a la información de la cuenta")

	_MsgAccNotFound = "Cuenta no encontrada"
  _MsgAccCreated = "Registro exitoso"
	_MsgLoggedIn = "Inicio de sesión exitoso"
	_MsgAccDeactivated = "La cuenta fue desactivada"
	_MsgUsernameChanged = "Se cambió el nombre de usuario"
	_MsgPasswordChanged = "Se cambió la contraseña"
)

func checkLoginAttempts(w http.ResponseWriter, r *http.Request) (fp models.Fingerprint, err error) {
  fp = r.Context().Value(models.FINGERPRINT).(models.Fingerprint)
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
