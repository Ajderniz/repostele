package controllers

import (
	"net/http"

	"github.com/ajderniz/repostele/pkg/bind"
)

const (
  _CREDS_VALIDATE     = "required,min=4,max=16,alphanum"
  
  _CREDS_USERNAME     = "username"
  _CREDS_NEW_USERNAME = "new-username"
  _CREDS_PASSWORD     = "password"
  _CREDS_OLD_PASSWORD = "old-password"
  _CREDS_NEW_PASSWORD = "new-password"
)

func getCredsFromForm(r *http.Request) (username, password string, err error) {
  username, err = bind.FormValue(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil { return }
  password, err = bind.FormValue(r, _CREDS_PASSWORD, _CREDS_VALIDATE)
  return
}

func getNewUsernameFromForm(r *http.Request) (password, newUsername string, err error) {
  password, err = bind.FormValue(r, _CREDS_PASSWORD, _CREDS_VALIDATE)
  if err != nil { return }
  newUsername, err = bind.FormValue(r, _CREDS_NEW_USERNAME, _CREDS_VALIDATE)
  return
}

func getNewPasswordFromForm(r *http.Request) (oldPassword, newPassword string, err error) {
  oldPassword, err = bind.FormValue(r, _CREDS_OLD_PASSWORD, _CREDS_VALIDATE)
  if err != nil { return }
  newPassword, err = bind.FormValue(r, _CREDS_NEW_PASSWORD, _CREDS_VALIDATE)
  return
}