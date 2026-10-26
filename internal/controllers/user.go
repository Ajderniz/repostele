package controllers

import (
	"net/http"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
)

func deactivateUserAccount(
  w http.ResponseWriter,
  r *http.Request,
  username string,
  self bool,
) (int, error) {

  if status, err := closeAccountSessions(w, r, username, models.SESSION_ROLE_USER, self); err != nil {
    return status, err
  }

  err := models.UpdateUserField(username, models.USER_ACTIVE, false)
  if err != nil { return InternalServerError, err }

  return OK, nil
}

func DeactivateUserAccount(w http.ResponseWriter, r *http.Request) {
  username, err := bind.URLParam(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil { serveBadRequestHX(w, err.Error()); return }

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

  cell := map[string]any{"Username": username, "Active": false, "OOB": true}
  serveOOB(w, OK, _HXData{Msg: _MsgAccDeactivated},
    frag("user-active-cell", cell), frag("user-actions-cell", cell))
}

func GetUserList(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { serveBadRequest(w, r, err); return }

  users, hasNext, err := models.GetUsers(&params)
  if err != nil { serveInternalErr(w, r); return }

  serveDataHX(w, r, map[string]any{
    "Users":   users,
    "Params":  params,
    "HasNext": hasNext,
  }, "list-users")
}
