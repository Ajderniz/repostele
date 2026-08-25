package controllers

import (
	"net/http"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
	"github.com/ajderniz/repostele/pkg/pass"
)

func updateUserPassword(username, oldPassword, newPassword string) (int, error){
  if oldPassword == newPassword {return BadRequest, _ErrSamePassword}

  user, err := models.GetUserFromUsername(username)
  if err != nil || user.Username == "" {return InternalServerError,err}

  err = pass.CheckPasswordHash(oldPassword, user.PassHash)
  if err != nil { return Unauthorized, err }

  newHash, err := pass.HashPassword(newPassword)
  if err != nil { return InternalServerError, err }

  err = models.UpdateUserField(username, models.USER_PASS_HASH, newHash)
  if err != nil { return InternalServerError, err }

  return OK, nil
}

func UpdateUserPassword(w http.ResponseWriter, r *http.Request) {
  username, err := bind.FormValue(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  oldPassword, newPassword, err := getNewPasswordFromForm(r)
  if err != nil { serveBadRequest(w, r, err); return }

  status, err := updateUserPassword(username, oldPassword, newPassword)
  if err != nil { serveErr(w, r, status, err); return }

  serveMsg(w, r, _MsgPasswordChanged)
}

func deactivateUserAccount(
  w http.ResponseWriter,
  r *http.Request,
  username string,
  self bool,
) (int, error) {

  if self {
    sid, err := r.Cookie(SESSION_ID)
    if err != nil { return BadRequest, err }
    err = closeSession(w, sid.Value)
    if err != nil { return InternalServerError, err }
  } else {
    if err := models.CloseSessionForUsername(username); err != nil {
      return InternalServerError, err
    }
  }

  err := models.UpdateUserField(username, models.USER_ACTIVE, false)
  if err != nil { return InternalServerError, err }

  return OK, nil
}

func DeactivateUserAccount(w http.ResponseWriter, r *http.Request) {
  username, err := bind.FormValue(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil { serveBadRequest(w, r, err); return }

  nextAction := _NextAction{
    URL: "/dashboard/admin/users",
    Name: "Volver a la lista de usuarios",
    HTMX: true,
  }

  status, err := deactivateUserAccount(w, r, username, false)
  if err != nil {
    serveResponseHX(w, err.Error(), status, &nextAction)
    return
}

  serveResponseHX(w,_MsgAccDeactivated, OK, &nextAction)
}

func GetUserList(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { serveBadRequest(w, r, err); return }

  users, err := models.GetUsers(params)
  if err != nil { serveInternalErr(w, r); return }

  serveDataHX(w, r, users, "list-users")
}

func GetUserFromUsername(w http.ResponseWriter, r *http.Request) {
  username, err := bind.URLParam(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil { serveInternalErr(w, r); return }

  user, err := models.GetUserFromUsername(username)
  if err != nil { serveInternalErr(w, r); return }

  if user.Username == "" { serveNoResults(w, r); return }
  user.PassHash = ""
  serveData(w, r, user)
}
