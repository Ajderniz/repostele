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

func SelfRegisterAccount(w http.ResponseWriter, r *http.Request) {
  fp := r.Context().Value(models.FINGERPRINT).(models.Fingerprint)
  if _MAX_REG_ACCS <= fp.AccsCreated {
    serveErr(w, r, Forbidden, errors.New("Account creation limit reached"))
    return
  }

  username, password, err := getCredsFromForm(r)
  if err != nil { serveBadRequest(w, r, err); return }

  user := models.User{}
  user.Username = username
  user.PassHash, err = pass.HashPassword(password)
  if err != nil { serveInternalErr(w, r); return }
  user.TimeCreated = time.Now().Unix()

  err = models.InsertUserAccount(user, fp)
  if err != nil { serveInternalErr(w, r); return }

  serveResponse(w, r, &_MainData{Msg: _MsgAccCreated}, Created, nil)
}

func Login(w http.ResponseWriter, r *http.Request) {
  fp, err := checkLoginAttempts(w, r)
  if err != nil { serveErr(w, r, Forbidden, err); return }

  username, password, err := getCredsFromForm(r)
  if err != nil { serveBadRequest(w, r, err); return }

  user, err := models.GetUserFromUsername(username)
  if err != nil { serveInternalErr(w, r); return }
  if user.Username == "" { serveMsg(w, r, _MsgAccNotFound); return }

  err = pass.CheckPasswordHash(password, user.PassHash)
  if err != nil { failLogin(w, r, fp); return }

  sessionID, _ := r.Cookie(SESSION_ID)

  err = openSession(
    w,
    user.Username,
    models.SESSION_ROLE_USER,
    sessionID.Value,
    fp.Id,
  )
  if err != nil { serveInternalErr(w, r); return}

  serveData(w, r, &_MainData{Data: user.Username})
}

func SelfUpdatePassword(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(_CREDS_USERNAME).(string)
  oldPassword, newPassword, err := getNewPasswordFromForm(r)
  if err != nil { serveBadRequest(w, r, err); return }

  status, err := updateUserPassword(username, oldPassword, newPassword)
  if err != nil { serveErr(w, r, status, err); return }

  serveMsg(w, r, _MsgPasswordChanged)
}

func SelfDeactivateAccount(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(_CREDS_USERNAME).(string)
  password, err := bind.FormValue(r, _CREDS_PASSWORD, _CREDS_VALIDATE)
  if err != nil { serveInternalErr(w, r); return }

  user, err := models.GetUserFromUsername(username)
  if err != nil || user.Username == "" { serveInternalErr(w, r); return }

  err = pass.CheckPasswordHash(password, user.PassHash)
  if err != nil { serveErr(w, r, Unauthorized, err) }

  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil { serveInternalErr(w, r); return }
  if latestOrder.RefNum != "" &&
     latestOrder.Status != models.ORDER_STATUS_CANCELLED &&
     latestOrder.Status != models.ORDER_STATUS_FULFILLED {
    serveErr(w, r, Conflict, errors.New("An order is pending"))
    return
  }

  status, err := deactivateUserAccount(w, r, user.Username)
  if err != nil { serveErr(w, r, status, err); return }

  serveMsg(w, r, _MsgAccDeactivated)
}
