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

const _TOKEN_LENGTH = 32

func RegisterUserAccount(w http.ResponseWriter, r *http.Request) {
  username, password, err := getCredsFromForm(r)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  user := models.User{}
  user.Username = username
  user.PassHash, err = pass.HashPassword(password)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}
  user.TimeCreated = time.Now().Unix()

  err = models.InsertUserAccount(user)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}

  write.Msg(w, _MsgAccCreated)
}

func UserLogin(w http.ResponseWriter, r *http.Request) {
  closeSession(w, r) // don't check for errors

  username, password, err := getCredsFromForm(r)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  user, err := models.GetUserFromUsername(username)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}
  if user.Username == "" { write.Msg(w, _MsgAccNotFound); return }

  err = pass.CheckPasswordHash(password, user.PassHash)
  if err != nil { write.Error(w, http.StatusUnauthorized, err); return }

  err = openSession(w, user.Username, models.SESSION_ROLE_USER)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}

  write.Data(w, write.H{
    write.KEY_DAT: write.H{
      models.USER_USERNAME: user.Username,
      models.USER_TIME_CREATED: user.TimeCreated,
    },
  })
}

func UpdateUserUsername(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(_CREDS_USERNAME).(string)
  password, newUsername, err := getNewUsernameFromForm(r)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  user, err := models.GetUserFromUsername(username)
  if err != nil || user.Username == "" {
    write.Error(w, http.StatusInternalServerError, err)
    return
  }

  if username == newUsername {
    write.Error(w, http.StatusConflict, _ErrSameUsername)
    return
  }

  err = pass.CheckPasswordHash(password, user.PassHash)
  if err != nil { write.Error(w, http.StatusUnauthorized, err); return }

  err = models.UpdateUserField(user.Username, _CREDS_USERNAME, newUsername)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}

  write.Msg(w, _MsgUsernameChanged)
}

func updateUserPassword(username, oldPassword, newPassword string) (int, error){
  if oldPassword == newPassword {return http.StatusBadRequest, _ErrSamePassword}

  user, err := models.GetUserFromUsername(username)
  if err != nil||user.Username == "" {return http.StatusInternalServerError,err}

  err = pass.CheckPasswordHash(oldPassword, user.PassHash)
  if err != nil { return http.StatusUnauthorized, err }

  newHash, err := pass.HashPassword(newPassword)
  if err != nil { return http.StatusInternalServerError, err }

  err = models.UpdateUserField(username, models.USER_PASS_HASH, newHash)
  if err != nil { return http.StatusInternalServerError, err }

  return http.StatusOK, nil
}

func UpdateUserPassword(w http.ResponseWriter, r *http.Request) {
  username := ""
  err := bind.FormValue(r, &username, _CREDS_USERNAME, _CREDS_VALIDATE)
  oldPassword, newPassword, err := getNewPasswordFromForm(r)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  status, err := updateUserPassword(username, oldPassword, newPassword)
  if err != nil { write.Error(w, status, err ); return }

  write.Msg(w, _MsgPasswordChanged)
}

func UpdateUserPasswordSelf(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(_CREDS_USERNAME).(string)
  oldPassword, newPassword, err := getNewPasswordFromForm(r)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  status, err := updateUserPassword(username, oldPassword, newPassword)
  if err != nil { write.Error(w, status, err ); return }

  write.Msg(w, _MsgPasswordChanged)
}

func deactivateUserAccount(w http.ResponseWriter, r *http.Request,
                           username string) (int, error) {
  err := closeSession(w, r)
  if err != nil { return http.StatusInternalServerError, err }

  err = models.UpdateUserField(username, models.USER_ACTIVE, false)
  if err != nil { return http.StatusInternalServerError, err }

  return http.StatusOK, nil
}

func DeactivateUserAccount(w http.ResponseWriter, r *http.Request) {
  username := ""
  err := bind.FormValue(r, username, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  status, err := deactivateUserAccount(w, r, username)
  if err != nil { write.Error(w, status, err); return }

  write.Msg(w, _MsgAccDeactivated)
}

func DeactivateUserAccountSelf(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(_CREDS_USERNAME).(string)
  password := ""
  err := bind.FormValue(r, &password, _CREDS_PASSWORD, _CREDS_VALIDATE)
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

func GetUserList(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  users, err := models.GetUsers(params)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}

  if users == nil { write.Data(w, _DataNoResults); return }
  write.Data(w, users)
}

func GetUserFromUsername(w http.ResponseWriter, r *http.Request) {
  username := ""
  err := bind.FormValue(r, &username, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}

  user, err := models.GetUserFromUsername(username)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}

  if user.Username == "" { write.Data(w, _DataNoResults) }
  write.Data(w, user)
}