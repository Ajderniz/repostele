//go:build STAFF

package controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
	"github.com/ajderniz/repostele/pkg/pass"
)

const _TEST_KEY = "1111111111111111"

func makeNewStaffFromForm(r *http.Request) (models.Staff, int, error) {
  username, err := bind.FormValue(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil { return models.Staff{}, BadRequest, err }
  password1, err := bind.FormValue(r, _CREDS_PASSWORD+"-1", _CREDS_VALIDATE)
  if err != nil { return models.Staff{}, BadRequest, err }
  password2, err := bind.FormValue(r, _CREDS_PASSWORD+"-2", _CREDS_VALIDATE)
  if err != nil {
    return models.Staff{}, BadRequest, err
  }
  if password1 != password2 {
    return models.Staff{}, BadRequest, errors.New("Password missmatch")
  }
  fullName, err := bind.FormValue(r, "full-name", "required,alphanumspace")
  if err != nil { return models.Staff{}, BadRequest, err }

  staff := models.Staff{}
  staff.Username = username
  staff.PassHash, err = pass.HashPassword(password1)
  if err != nil { return models.Staff{}, InternalServerError, err }
  staff.FullName = fullName
  staff.TimeCreated = time.Now().Unix()
  staff.Admin = false

  return staff, OK, nil
}

func InitMainStaffAccount(w http.ResponseWriter, r *http.Request) {
  list, err := models.GetStaff(models.SelectParams{
    Start: 0, Limit: 1, Sort: models.USER_USERNAME, Dir: models.SORT_DIR_ASC,
  })
  if 1 <= len(list) {
    slog.Error(_ErrAlreadyInit.Error())
    http.Redirect(w, r, "/", MovedPermanently)
    return
  }

  key, err := bind.FormValue(r, "key", "required,number,len=16")
  if err != nil { serveBadRequest(w, r, err); return }
  if key != _TEST_KEY {
    serveErr(w, r, Unauthorized, errors.New("Clave de activación inválida"))
    return
  }

  staff, status, err := makeNewStaffFromForm(r)
  if err != nil { serveErr(w, r, status, err); return }

  staff.Admin = true

  err = models.InsertStaffAccount(staff)
  if err != nil { serveInternalErr(w, r); return }

  serveResponse(w, r, &_MainData{Msg: _MsgAccCreated}, Created, nil)
}

func RegisterStaffAccount(w http.ResponseWriter, r *http.Request) {
  staff, status, err := makeNewStaffFromForm(r)
  if err != nil { serveResponseHX(w, err.Error(), status, nil); return }

  adminStr, err := bind.FormValue(r, "admin", "required,boolean")
  if err != nil { serveBadRequestHX(w, "Parámetro 'admin' inválido"); return }
  staff.Admin, _ = strconv.ParseBool(adminStr)

  err = models.InsertStaffAccount(staff)
  if err != nil { serveInternalErrHX(w); return }
  
  serveResponseHX(w, _MsgAccCreated, Created, &_NextAction{
    URL: "/htmx/form-register-staff-admin",
    Name: "Agregar otro",
    HTMX: true,
  })
}

func Login(w http.ResponseWriter, r *http.Request) {
  fp, err := checkLoginAttempts(w, r)
  if err != nil { serveErr(w, r, Forbidden, err); return }

  username, password, err := getCredsFromForm(r)
  if err != nil { serveBadRequest(w, r, err); return }

  staff, err := models.GetStaffFromUsername(username)
  if err != nil { serveInternalErr(w, r); return }
  if staff.Username == "" { serveMsg(w, r, _MsgAccNotFound); return }

  err = pass.CheckPasswordHash(password, staff.PassHash)
  if err != nil { failLogin(w, r, fp); return }

  sessionIDCookie, _ := r.Cookie(SESSION_ID)
  sessionID := ""
  if sessionIDCookie != nil { sessionID = sessionIDCookie.Value }

  err = openSession(
    w, 
    staff.Username, 
    models.SESSION_ROLE_STAFF,
    sessionID,
    fp.Id,
  )
  if err != nil { serveInternalErr(w, r); return }

  serveMsg(w, r, _MsgLoggedIn)
}

func deactivateStaffAccount(
  w http.ResponseWriter,
  r *http.Request,
  username string,
  self bool,
) (int, error) {

  target, err := models.GetStaffFromUsername(username)
  if err != nil { return InternalServerError, err }

  if target.Admin {
    admins, err := models.GetStaffAdmins()
    if err != nil { return InternalServerError, err }
    if len(admins) <= 1 {
      return Forbidden, errors.New(
        "Es necesaria al menos una cuenta de administrador activa",
      )
    }
  }

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

  err = models.UpdateStaffField(username, models.STAFF_ACTIVE, false)
  if err != nil { return InternalServerError, err }

  return OK, nil
}

func DeactivateStaffAccount(w http.ResponseWriter, r *http.Request) {
  username, err := bind.FormValue(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil { serveBadRequestHX(w, err.Error()); return }

  status, err := deactivateStaffAccount(w, r, username, false)
  if err != nil { serveResponseHX(w, err.Error(), status, nil); return }

  serveResponseHX(w, _MsgAccDeactivated, OK, nil)
}

func SelfDeactivateAccount(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(_CREDS_USERNAME).(string)
  password, err := bind.FormValue(r, _CREDS_PASSWORD, _CREDS_VALIDATE)
  if err != nil { serveInternalErr(w, r); return }

  staff, err := models.GetStaffFromUsername(username)
  if err != nil || staff.Username == "" { serveInternalErr(w, r); return }

  err = pass.CheckPasswordHash(password, staff.PassHash)
  if err != nil { serveErr(w, r, Unauthorized, err); return }

  status, err := deactivateStaffAccount(w, r, username, true)
  if err != nil { serveErr(w, r, status, err); return }

  serveMsg(w, r, _MsgAccDeactivated)
}

func updateStaffPasssword(username, oldPassword, newPassword string)(int,error){
  if oldPassword == newPassword {return BadRequest, _ErrSamePassword}

  staff, err := models.GetStaffFromUsername(username)
  if err != nil||staff.Username == ""{return InternalServerError,err}

  err = pass.CheckPasswordHash(oldPassword, staff.PassHash)
  if err != nil { return Unauthorized, err }

  newHash, err := pass.HashPassword(newPassword)
  if err != nil { return InternalServerError, err }

  err = models.UpdateStaffField(username, models.STAFF_PASS_HASH, newHash)
  if err != nil { return InternalServerError, err }

  return OK, nil
}

func UpdateStaffPassword(w http.ResponseWriter, r *http.Request) {
  username, err := bind.FormValue(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  oldPassword, newPassword, err := getNewPasswordFromForm(r)
  if err != nil { serveBadRequest(w, r, err); return }

  status, err := updateStaffPasssword(username, oldPassword, newPassword)
  if err != nil { serveErr(w, r, status, err); return }

  serveMsg(w, r, _MsgPasswordChanged)
}

func SelfUpdatePassword(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)
  oldPassword, newPassword, err := getNewPasswordFromForm(r)
  if err != nil { serveBadRequest(w, r, err); return }

  status, err := updateStaffPasssword(username, oldPassword, newPassword)
  if err != nil { serveErr(w, r, status, err); return }

  serveMsg(w, r, _MsgPasswordChanged)
}

func GetStaffList(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { serveBadRequest(w, r, err); return }

  staff, err := models.GetStaff(params)
  if err != nil { serveInternalErr(w, r); return }

  serveDataHX(w, r, staff, "list-staff")
}

func GetStaffFromUsername(w http.ResponseWriter, r *http.Request) {
  username, err := bind.URLParam(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil { serveBadRequest(w, r, err); return }

  staff, err := models.GetStaffFromUsername(username)
  if err != nil { serveInternalErr(w, r); return }

  if staff.Username == "" { serveNoResults(w, r); return }
  staff.PassHash = ""
  serveData(w, r, staff)
}
