//go:build STAFF

package controllers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ajderniz/repostele/internal/models"

	"github.com/ajderniz/repostele/internal/reqlog"
	"github.com/ajderniz/repostele/pkg/bind"
	"github.com/ajderniz/repostele/pkg/pass"
)

const (
  _SERVER_NAME = "Staff"
  _TEST_KEY = "1111111111111111"
)

var _ErrAlreadyInit = errors.New("El sistema ya está configurado")

func checkInit(
  w http.ResponseWriter,
  r *http.Request,
  section string,
) (init, redirect bool) {
  init = true
  if section == "init" {
    if models.CheckInit() {
      http.Redirect(w, r, "/menu", MovedPermanently)
      redirect = true
      return
    }
    init = false
  }
  return
}

func HandleRoot(w http.ResponseWriter, r *http.Request) {
  if !models.CheckInit() {
    http.Redirect(w, r, "/init", PermanentRedirect)
  } else {
    http.Redirect(w, r, "/menu", PermanentRedirect)
  }
}

func GetDashboard(w http.ResponseWriter, r *http.Request) {
  stats := getDashStats(r)
  ctx := context.WithValue(r.Context(), _MAIN_DATA, &_MainData{Stats: &stats})
  ServeMainTemplate(w, r.WithContext(ctx))
}

func startOfToday() int64 {
  now := time.Now()
  return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
}


func GetQueueCounts(w http.ResponseWriter, r *http.Request) {
  c, err := models.GetQueueCounts(startOfToday())
  if err != nil { reqlog.Error(r, err) }
  serveDataHX(w, r, c, "queue-counts")
}

func GetDashTiles(w http.ResponseWriter, r *http.Request) {
  serveDataHX(w, r, getDashStats(r), "dash-tiles")
}

func getDashStats(r *http.Request) _DashStats {
  var stats _DashStats
  day, err := models.GetDayStats(startOfToday())
  if err != nil { reqlog.Error(r, err) }
  stats.OrdersToday, stats.SalesToday = day.Orders, day.Sales
  queue, err := models.GetPendingQueueStats()
  if err != nil { reqlog.Error(r, err) } else { stats.InQueue = queue.Count }
  return stats
}

func makeNewStaffFromForm(r *http.Request) (models.Staff, int, error) {
  username, password, err := getRegisterCredsFromForm(r)
  if err != nil { return models.Staff{}, BadRequest, err }
  fullName, err := bind.FormValue(r, "full-name", "required,alphanumspace")
  if err != nil { return models.Staff{}, BadRequest, err }

  staff := models.Staff{}
  staff.Username = username
  staff.PassHash, err = pass.HashPassword(password)
  if err != nil { return models.Staff{}, InternalServerError, err }
  staff.FullName = fullName
  staff.TimeCreated = time.Now().Unix()
  staff.Admin = false

  return staff, OK, nil
}

func InitMainStaffAccount(w http.ResponseWriter, r *http.Request) {
  list, _, err := models.GetStaff(&models.SelectParams{
    Start: 0, Limit: 1, Sort: models.USER_USERNAME, Dir: models.SORT_DIR_ASC,
  })
  if err != nil { serveInternalErr(w, r, err); return }
  if 1 <= len(list) {
    reqlog.Error(r, _ErrAlreadyInit)
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
  if err != nil { serveInternalErr(w, r, err); return }

  serveResponse(w, r, &_MainData{Msg: _MsgAccCreated}, Created, nil)
}

func RegisterStaffAccount(w http.ResponseWriter, r *http.Request) {
  staff, status, err := makeNewStaffFromForm(r)
  if err != nil { serveStatusErrHX(w, r, status, err); return }

  adminStr, err := bind.FormValue(r, "admin", "required,boolean")
  if err != nil { serveBadRequestHX(w, "Parámetro 'admin' inválido"); return }
  staff.Admin, _ = strconv.ParseBool(adminStr)

  err = models.InsertStaffAccount(staff)
  if err != nil { serveInternalErrHX(w, r, err); return }
  
  serveResponseHX(w, _MsgAccCreated, Created, &_NextAction{
    URL: "/htmx/form-register-staff-admin",
    Name: "Agregar otro",
    HTMX: true,
  })
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

  if status, err := closeAccountSessions(w, r, username, models.SESSION_ROLE_STAFF, self); err != nil {
    return status, err
  }

  err = models.UpdateStaffField(username, models.STAFF_ACTIVE, false)
  if err != nil { return InternalServerError, err }

  return OK, nil
}

func DeactivateStaffAccount(w http.ResponseWriter, r *http.Request) {
  username, err := bind.URLParam(r, _CREDS_USERNAME, _CREDS_VALIDATE)
  if err != nil { serveBadRequestHX(w, err.Error()); return }

  status, err := deactivateStaffAccount(w, r, username, false)
  if err != nil { serveStatusErrHX(w, r, status, err); return }

  cell := map[string]any{"Username": username, "Active": false, "OOB": true}
  serveOOB(w, OK, _HXData{Msg: _MsgAccDeactivated},
    frag("staff-active-cell", cell), frag("staff-actions-cell", cell))
}

func GetStaffList(w http.ResponseWriter, r *http.Request) {
  params := models.SelectParams{}
  err := bind.Form(r, &params)
  if err != nil { serveBadRequest(w, r, err); return }

  staff, hasNext, err := models.GetStaff(&params)
  if err != nil { serveInternalErr(w, r, err); return }

  serveDataHX(w, r, map[string]any{
    "Staff":   staff,
    "Params":  params,
    "HasNext": hasNext,
  }, "list-staff")
}

func accountPassHash(username string) (string, error) {
  staff, err := models.GetStaffFromUsername(username)
  return staff.PassHash, err
}

func setAccountPassHash(username, hash string) error {
  return models.UpdateStaffField(username, models.STAFF_PASS_HASH, hash)
}

func selfDeactivate(w http.ResponseWriter, r *http.Request, username string) (int, error) {
  return deactivateStaffAccount(w, r, username, true)
}
