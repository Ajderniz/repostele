//go:build USER

package controllers

import (
	"errors"
	"net/http"
	"time"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/pass"
)

const _SERVER_NAME = "User"

func checkInit(
	w http.ResponseWriter,
	r *http.Request,
	s string,
) (init, redirect bool) {
	return true, false
}

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/menu", PermanentRedirect)
}

// registerFormErr re-renders the register form with the error above it,
// since the form's own hx-target is #login-panel (not a shared *HX helper's
// div-response slot) — losing the fields on error would be a worse UX than
// the extra render.
func registerFormErr(w http.ResponseWriter, r *http.Request, status int, err error) {
  if isHX(r) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(status)
    _Tpl.ExecuteTemplate(w, "div-msg", err.Error())
    _Tpl.ExecuteTemplate(w, "form-register-user", nil)
    return
  }
  serveErr(w, r, status, err)
}

func SelfRegisterAccount(w http.ResponseWriter, r *http.Request) {
  fp := models.FingerprintFromContext(r.Context())
  if _MAX_REG_ACCS <= fp.AccsCreated {
    registerFormErr(w, r, Forbidden, errors.New("Se alcanzó el límite de creación de cuentas"))
    return
  }

  username, password, err := getRegisterCredsFromForm(r)
  if err != nil { registerFormErr(w, r, BadRequest, err); return }

  user := models.User{}
  user.Username = username
  user.PassHash, err = pass.HashPassword(password)
  if err != nil { registerFormErr(w, r, InternalServerError, _ErrInternal); return }
  user.TimeCreated = time.Now().Unix()

  err = models.InsertUserAccount(user, fp)
  if err != nil { registerFormErr(w, r, InternalServerError, err); return }

  if isHX(r) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(Created)
    _Tpl.ExecuteTemplate(w, "div-msg", _MsgAccCreated)
    _Tpl.ExecuteTemplate(w, "form-login", nil)
    return
  }
  serveResponse(w, r, &_MainData{Msg: _MsgAccCreated}, Created, nil)
}

func accountPassHash(username string) (string, error) {
  user, err := models.GetUserFromUsername(username)
  return user.PassHash, err
}

func setAccountPassHash(username, hash string) error {
  return models.UpdateUserField(username, models.USER_PASS_HASH, hash)
}

// Customers can't leave with an order still open.
func selfDeactivate(w http.ResponseWriter, r *http.Request, username string) (int, error) {
  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil { return InternalServerError, _ErrInternal }
  if latestOrder.Pending() { return Conflict, errors.New("Hay una orden pendiente") }
  return deactivateUserAccount(w, r, username, true)
}
