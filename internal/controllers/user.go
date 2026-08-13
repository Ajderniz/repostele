package controllers

import (
	"net/http"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
	"github.com/ajderniz/repostele/pkg/pass"
	"github.com/ajderniz/repostele/pkg/write"
)

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
  username, err := bind.FormValue(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  oldPassword, newPassword, err := getNewPasswordFromForm(r)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  status, err := updateUserPassword(username, oldPassword, newPassword)
  if err != nil { write.Error(w, status, err ); return }

  write.Msg(w, _MsgPasswordChanged)
}

func deactivateUserAccount(w http.ResponseWriter, r *http.Request,
                           username string) (int, error) {

  sid, err := r.Cookie(SESSION_ID)
  if err != nil { return http.StatusBadRequest, err }

  err = closeSession(w, sid.Value)
  if err != nil { return http.StatusInternalServerError, err }

  err = models.UpdateUserField(username, models.USER_ACTIVE, false)
  if err != nil { return http.StatusInternalServerError, err }

  return http.StatusOK, nil
}

func DeactivateUserAccount(w http.ResponseWriter, r *http.Request) {
  username, err := bind.FormValue(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  status, err := deactivateUserAccount(w, r, username)
  if err != nil { write.Error(w, status, err); return }

  write.Msg(w, _MsgAccDeactivated)
}

func GetUserList(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  users, err := models.GetUsers(params)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}

  if len(users) <= 0 { write.Data(w, _MsgNoResults); return }
  write.Data(w, users)
}

func GetUserFromUsername(w http.ResponseWriter, r *http.Request) {
  username, err := bind.URLParam(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil {write.Error(w, http.StatusInternalServerError, err); return }

  user, err := models.GetUserFromUsername(username)
  if err != nil {write.Error(w, http.StatusInternalServerError, err); return }

  if user.Username == "" { write.Data(w, _MsgNoResults); return }
  user.PassHash = ""
  write.Data(w, user)
}
