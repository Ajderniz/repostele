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
	"github.com/ajderniz/repostele/pkg/write"
)

const _TEST_KEY = "1111111111111111"

func makeNewStaffFromForm(r *http.Request) (models.Staff, int, error) {
  username, err := bind.FormValue(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil { return models.Staff{}, http.StatusBadRequest, err }
  password1, err := bind.FormValue(r, _CREDS_PASSWORD+"-1", _CREDS_VALIDATE)
  if err != nil { return models.Staff{}, http.StatusBadRequest, err }
  password2, err := bind.FormValue(r, _CREDS_PASSWORD+"-2", _CREDS_VALIDATE)
  if err != nil {
    return models.Staff{}, http.StatusBadRequest, err
  }
  if password1 != password2 {
    slog.Error("BAD BAD BAD")
    return models.Staff{}, http.StatusBadRequest, errors.New("Password missmatch")
  }
  fullName, err := bind.FormValue(r, "full-name", "required,alphanumspace")
  if err != nil { return models.Staff{}, http.StatusBadRequest, err }


  staff := models.Staff{}
  staff.Username = username
  staff.PassHash, err = pass.HashPassword(password1)
  if err != nil { return models.Staff{}, http.StatusInternalServerError, err }
  staff.FullName = fullName
  staff.TimeCreated = time.Now().Unix()
  staff.Admin = false

  return staff, http.StatusOK, nil
}

func InitMainStaffAccount(w http.ResponseWriter, r *http.Request) {
  list, err := models.GetStaff(models.SelectParams{
    Start: 0, Limit: 1, Sort: models.USER_USERNAME, Dir: models.SORT_DIR_ASC,
  })
  if 1 <= len(list) {
    slog.Error(_ErrAlreadyInit.Error())
    write.Error(w, http.StatusForbidden, _ErrAlreadyInit)
    return
  }

  key, err := bind.FormValue(r, "key", "required,number,len=16")
  if err != nil || key != _TEST_KEY {
    w.WriteHeader(http.StatusUnauthorized)
    return
  }

  staff, status, err := makeNewStaffFromForm(r)
  if err != nil { write.Error(w, status, err); return }

  staff.Admin = true

  err = models.InsertStaffAccount(staff)
  if err != nil { write.Error(w, http.StatusInternalServerError, err); return }

  write.Msg(w, _MsgAccCreated)
}

func RegisterStaffAccount(w http.ResponseWriter, r *http.Request) {
  staff, status, err := makeNewStaffFromForm(r)
  if err != nil { write.Error(w, status, err); return }

  adminStr, err := bind.FormValue(r, "admin", "required,boolean")
  if err != nil {
    write.Error(w, http.StatusBadRequest,errors. New("'admin' tag invalid"))
    return
  }
  staff.Admin, _ = strconv.ParseBool(adminStr)

  err = models.InsertStaffAccount(staff)
  if err != nil { write.Error(w, http.StatusInternalServerError, err); return }
  
  write.Msg(w, _MsgAccCreated)
}

func StaffLogin(w http.ResponseWriter, r *http.Request) {
  fp, err := checkLoginAttempts(w, r)
  if err != nil { write.Error(w, http.StatusForbidden, err); return }

  username, password, err := getCredsFromForm(r)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  staff, err := models.GetStaffFromUsername(username)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}
  if staff.Username == "" { write.Msg(w, _MsgAccNotFound); return }

  err = pass.CheckPasswordHash(password, staff.PassHash)
  if err != nil { failLogin(w, fp); return }

  sessionID, _ := r.Cookie(SESSION_ID)

  err = openSession(
    w, 
    staff.Username, 
    models.SESSION_ROLE_STAFF,
    sessionID.Value,
    fp.Id,
  )
  if err != nil { write.Error(w, http.StatusInternalServerError, err); return }

  write.Data(w, write.H{
    models.STAFF_USERNAME: staff.Username,
    models.STAFF_TIME_CREATED: staff.TimeCreated,
  })
}

func deactivateStaffAccount(
  w http.ResponseWriter,
  r *http.Request,
  username string,
) (int, error) {
  
  admins, err := models.GetStaffAdmins()
  if err != nil { return http.StatusInternalServerError, err }
  if len(admins) <= 1 { 
    return http.StatusForbidden,
           errors.New("At least one admin account is required to be active")
  }

  sid, err := r.Cookie(SESSION_ID)
  if err != nil { return http.StatusBadRequest, err }

  err = closeSession(w, sid.Value)
  if err != nil { return http.StatusInternalServerError, err }

  err = models.UpdateStaffField(username, models.STAFF_ACTIVE, false)
  if err != nil { return http.StatusInternalServerError, err }

  return http.StatusOK, nil
}

func DeactivateStaffAccount(w http.ResponseWriter, r *http.Request) {
  username, err := bind.FormValue(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  status, err := deactivateStaffAccount(w, r, username)
  if err != nil { write.Error(w, status, err); return }

  write.Msg(w, _MsgAccDeactivated)
}

func SelfDeactivateStaffAccount(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(_CREDS_USERNAME).(string)
  password, err := bind.FormValue(r, _CREDS_PASSWORD, _CREDS_VALIDATE)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}

  staff, err := models.GetStaffFromUsername(username)
  if err != nil || staff.Username == "" {
    write.Error(w, http.StatusInternalServerError, err)
    return
  }

  err = pass.CheckPasswordHash(password, staff.PassHash)
  if err != nil { write.Error(w, http.StatusUnauthorized, err); return }

  status, err := deactivateStaffAccount(w, r, username)
  if err != nil { write.Error(w, status, err); return }

  write.Msg(w, _MsgAccDeactivated)
}

func updateStaffPasssword(username, oldPassword, newPassword string)(int,error){
  if oldPassword == newPassword {return http.StatusBadRequest, _ErrSamePassword}

  staff, err := models.GetStaffFromUsername(username)
  if err != nil||staff.Username == ""{return http.StatusInternalServerError,err}

  err = pass.CheckPasswordHash(oldPassword, staff.PassHash)
  if err != nil { return http.StatusUnauthorized, err }

  newHash, err := pass.HashPassword(newPassword)
  if err != nil { return http.StatusInternalServerError, err }

  err = models.UpdateStaffField(username, models.STAFF_PASS_HASH, newHash)
  if err != nil { return http.StatusInternalServerError, err }

  return http.StatusOK, nil
}

func UpdateStaffPassword(w http.ResponseWriter, r *http.Request) {
  username, err := bind.FormValue(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  oldPassword, newPassword, err := getNewPasswordFromForm(r)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  status, err := updateStaffPasssword(username, oldPassword, newPassword)
  if err != nil { write.Error(w, status, err); return }

  write.Msg(w, _MsgPasswordChanged)
}

func SelfUpdateStaffPassword(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)
  oldPassword, newPassword, err := getNewPasswordFromForm(r)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  status, err := updateStaffPasssword(username, oldPassword, newPassword)
  if err != nil { write.Error(w, status, err); return }

  write.Msg(w, _MsgPasswordChanged)
}

func GetStaffList(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  staff, err := models.GetStaff(params)
  if err != nil {write.Error(w, http.StatusInternalServerError,err); return}

  if staff == nil { write.Data(w, _DataNoResults); return }

  write.Data(w, staff)
}

func GetStaffFromUsername(w http.ResponseWriter, r *http.Request) {
  username, err := bind.URLParam(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil { write.Error(w, http.StatusBadRequest, err); return }

  staff, err := models.GetStaffFromUsername(username)
  if err != nil {write.Error(w, http.StatusInternalServerError, err);return}

  if staff.Username == "" { write.Data(w, _DataNoResults); return }
  staff.PassHash = ""
  write.Data(w, staff)
}
