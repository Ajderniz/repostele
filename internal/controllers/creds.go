package controllers

import (
	"net/http"

	"github.com/ajderniz/repostele/pkg/bind"
)

const (
  _CREDS_VALIDATE     = "required,min=4,max=16,alphanum"
  
  _CREDS_USERNAME     = "username"
  _CREDS_NEW_USERNAME = "new_username"
  _CREDS_PASSWORD     = "password"
  _CREDS_OLD_PASSWORD = "old_password"
  _CREDS_NEW_PASSWORD = "new_password"
)

func getCredsFromForm(r *http.Request) (username, password string, err error) {
  err = bind.FormValue(r, &username, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil { return }
  err = bind.FormValue(r, &password, _CREDS_USERNAME, _CREDS_VALIDATE)
  return
}

func getNewUsernameFromForm(r *http.Request) (password, newUsername string, err error) {
  err = bind.FormValue(r, &password, _CREDS_PASSWORD, _CREDS_VALIDATE)
  if err != nil { return }
  err = bind.FormValue(r, &newUsername, _CREDS_NEW_USERNAME, _CREDS_VALIDATE)
  return
}

func getNewPasswordFromForm(r *http.Request) (oldPassword, newPassword string, err error) {
  err = bind.FormValue(r, &oldPassword, _CREDS_OLD_PASSWORD, _CREDS_VALIDATE)
  if err != nil { return }
  err = bind.FormValue(r, &newPassword, _CREDS_NEW_PASSWORD, _CREDS_VALIDATE)
  return
}