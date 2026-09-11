//go:build USER

package controllers

import (
	"errors"
	"net/http"
	"time"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
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
  if r.Header.Get("HX-Request") == "true" {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(status)
    _Tpl.ExecuteTemplate(w, "div-msg", err.Error())
    _Tpl.ExecuteTemplate(w, "form-register-user", nil)
    return
  }
  serveErr(w, r, status, err)
}

func SelfRegisterAccount(w http.ResponseWriter, r *http.Request) {
  fp := r.Context().Value(models.FINGERPRINT).(models.Fingerprint)
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

  if r.Header.Get("HX-Request") == "true" {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(Created)
    _Tpl.ExecuteTemplate(w, "div-msg", _MsgAccCreated)
    _Tpl.ExecuteTemplate(w, "form-login", nil)
    return
  }
  serveResponse(w, r, &_MainData{Msg: _MsgAccCreated}, Created, nil)
}

func UserLogin(w http.ResponseWriter, r *http.Request) {
  Login(w, r, false)
}

func SelfUpdatePassword(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(_CREDS_USERNAME).(string)
  oldPassword, newPassword, err := getNewPasswordFromForm(r)
  if err != nil { serveBadRequestHX(w, err.Error()); return }

  status, err := updateUserPassword(username, oldPassword, newPassword)
  if err != nil { serveResponseHX(w, err.Error(), status, nil); return }

  serveResponseHX(w, _MsgPasswordChanged, OK, nil)
}

func SelfDeactivateAccount(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(_CREDS_USERNAME).(string)
  password, err := bind.FormValue(r, _CREDS_PASSWORD, _CREDS_VALIDATE)
  if err != nil { serveInternalErrHX(w); return }

  user, err := models.GetUserFromUsername(username)
  if err != nil || user.Username == "" { serveInternalErrHX(w); return }

  err = pass.CheckPasswordHash(password, user.PassHash)
  if err != nil {
    serveResponseHX(w, _ErrBadCreds.Error(), Unauthorized, nil)
    return
  }

  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil { serveInternalErrHX(w); return }
  if latestOrder.RefNum != "" &&
     latestOrder.Status != models.ORDER_STATUS_CANCELLED &&
     latestOrder.Status != models.ORDER_STATUS_FULFILLED {
    serveResponseHX(w, "Hay una orden pendiente", Conflict, nil)
    return
  }

  status, err := deactivateUserAccount(w, r, user.Username, true)
  if err != nil { serveResponseHX(w, err.Error(), status, nil); return }

  if r.Header.Get("HX-Request") == "true" {
    w.Header().Set("HX-Redirect", "/login")
    w.WriteHeader(OK)
    return
  }
  serveMsg(w, r, _MsgAccDeactivated)
}
