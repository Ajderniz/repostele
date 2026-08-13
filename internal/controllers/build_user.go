//go:build USER

package controllers

import (
	"errors"
	"net/http"
	"time"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
	"github.com/ajderniz/repostele/pkg/pass"
	"github.com/ajderniz/repostele/pkg/write"
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
	http.Redirect(w, r, "/menu", http.StatusPermanentRedirect)
}

func SelfRegisterAccount(w http.ResponseWriter, r *http.Request) {
  fp := r.Context().Value(models.FINGERPRINT).(models.Fingerprint)
  if _MAX_REG_ACCS <= fp.AccsCreated {
    write.Error(w, http.StatusForbidden,
      errors.New("Account creation limit reached"),
    )
    return
  }

  username, password, err := getCredsFromForm(r)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  user := models.User{}
  user.Username = username
  user.PassHash, err = pass.HashPassword(password)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}
  user.TimeCreated = time.Now().Unix()

  err = models.InsertUserAccount(user, fp)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}

  write.Msg(w, _MsgAccCreated)
}

func Login(w http.ResponseWriter, r *http.Request) {
  fp, err := checkLoginAttempts(w, r)
  if err != nil { write.Error(w, http.StatusForbidden, err); return }

  username, password, err := getCredsFromForm(r)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  user, err := models.GetUserFromUsername(username)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}
  if user.Username == "" { write.Msg(w, _MsgAccNotFound); return }

  err = pass.CheckPasswordHash(password, user.PassHash)
  if err != nil { failLogin(w, fp); return }

  sessionID, _ := r.Cookie(SESSION_ID)

  err = openSession(
    w,
    user.Username,
    models.SESSION_ROLE_USER,
    sessionID.Value,
    fp.Id,
  )
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}

  write.Data(w, write.H{
    write.KEY_DAT: write.H{
      models.USER_USERNAME: user.Username,
      models.USER_TIME_CREATED: user.TimeCreated,
    },
  })
}

func SelfUpdatePassword(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(_CREDS_USERNAME).(string)
  oldPassword, newPassword, err := getNewPasswordFromForm(r)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  status, err := updateUserPassword(username, oldPassword, newPassword)
  if err != nil { write.Error(w, status, err ); return }

  write.Msg(w, _MsgPasswordChanged)
}

func SelfDeactivateAccount(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(_CREDS_USERNAME).(string)
  password, err := bind.FormValue(r, _CREDS_PASSWORD, _CREDS_VALIDATE)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}

  user, err := models.GetUserFromUsername(username)
  if err != nil || user.Username == "" {
    write.Error(w, http.StatusInternalServerError, err)
    return
  }

  err = pass.CheckPasswordHash(password, user.PassHash)
  if err != nil { write.Error(w, http.StatusUnauthorized, err); return}

  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}
  if latestOrder.RefNum != "" &&
     latestOrder.Status != models.ORDER_STATUS_CANCELLED &&
     latestOrder.Status != models.ORDER_STATUS_FULFILLED {
    write.Error(w, http.StatusConflict, errors.New("An order is pending."))
    return
  }

  status, err := deactivateUserAccount(w, r, user.Username)
  if err != nil { write.Error(w, status, err); return }

  write.Msg(w, _MsgAccDeactivated)
}
