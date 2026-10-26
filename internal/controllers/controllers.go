package controllers

import (
	"context"
	"errors"
	tpl "html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/static"
)

// Context keys for handing a response to ServeMainTemplate.
type _CtxKey int
const (
	_STATUS _CtxKey = iota
	_ERR
	_MAIN_DATA
)

const (
	OK = http.StatusOK
	Created = http.StatusCreated
	MovedPermanently = http.StatusMovedPermanently
	PermanentRedirect = http.StatusPermanentRedirect
	BadRequest = http.StatusBadRequest
	Unauthorized = http.StatusUnauthorized
	NotFound = http.StatusNotFound
	Forbidden = http.StatusForbidden
	Conflict = http.StatusConflict
	TooManyRequests = http.StatusTooManyRequests
	InternalServerError = http.StatusInternalServerError
)

type _MainData struct {
	Data          any
	Msg           string
	LoggedIn      bool
	IsStaff       bool
	IsAdmin       bool
	IsUserApp     bool
	Params        models.SelectParams
	HasNext       bool
	CurrentOrder  *models.Order
	Stats         *_DashStats
	Username      string
}

type _DashStats struct {
	OrdersToday int
	SalesToday  float32
	InQueue     int
}

type _TplData struct {
	Title    string
	Server   string
	Init     bool
	MainName string
	MainData *_MainData
	Err      string
}

type _NextAction struct {
	URL  string
	Name string
	HTMX bool
}

type _HXData struct {
	Msg         string
	NextAction  *_NextAction
}

var (
	_MsgEmpty = "Aún no hay nada"
  _MsgNoResults = "No se encontraron resultados"

  _ErrInternal = errors.New("Algo salió mal")
	_ErrBadSearch = errors.New("Criterios de búsqueda inválidos")

	_Tpl *tpl.Template

	titleCaser = cases.Title(language.Spanish)
)

// ServerName is the build's app name ("User" or "Staff").
func ServerName() string { return _SERVER_NAME }

func InitTemplate() {
	if err := loadAssetVersions(); err != nil { panic(err) }
	_Tpl = tpl.New("base")
	_Tpl.Funcs(tpl.FuncMap{
		"CallTemplate": callTemplate,
		"asset": asset,
		"OrderStatusName": orderStatusName,
		"UnixToTime": unixToTime,
		"GroupRef": groupRef,
		"dict": dict,
		"add": add,
		"subClamped": subClamped,
	})
	tpl.Must(_Tpl.ParseFS(static.FS, static.HTMDIR+"/*"))
	tpl.Must(_Tpl.ParseFS(static.FS, static.HXDIR+"/*"))
}

func ServeNav(w http.ResponseWriter, r *http.Request) {
	if !isHX(r) { http.NotFound(w, r); return }

	username, role, loggedIn := checkSessionUser(r)

	isStaff := false
	isAdmin := false
	pendingOrders := 0
	if loggedIn && role == models.SESSION_ROLE_STAFF {
		staff, err := models.GetStaffFromUsername(username)
		if err != nil {
			slog.Error(err.Error())
		} else if staff.Username != "" {
			isStaff = true
			isAdmin = staff.Admin
			stats, err := models.GetPendingQueueStats()
			if err != nil { slog.Error(err.Error()) } else { pendingOrders = stats.Unreviewed }
		}
	}

	// htmx sends the page URL the nav was loaded on; its first segment marks the active link.
	section := ""
	if cur, err := url.Parse(r.Header.Get("HX-Current-URL")); err == nil {
		section, _, _ = strings.Cut(strings.TrimPrefix(cur.Path, "/"), "/")
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := _Tpl.ExecuteTemplate(w, "site-nav", map[string]any{
		"Init":          models.CheckInit(),
		"LoggedIn":      loggedIn,
		"IsStaff":       isStaff,
		"IsAdmin":       isAdmin,
		"PendingOrders": pendingOrders,
		"Section":       section,
	})
	if err != nil { slog.Error(err.Error()); w.WriteHeader(InternalServerError) }
}

func ServeHTMX(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "path")
	if !isHX(r) || path == "" {
		http.NotFound(w, r); return
	}
	err := _Tpl.ExecuteTemplate(w, path, nil)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, _ErrInternal.Error(), InternalServerError);
	}
}

func ServeMainTemplate(w http.ResponseWriter, r *http.Request) {
	section := strings.Split(r.URL.Path, "/")[1]

	init, redirect := checkInit(w, r, section)
	if redirect { return }

	// Set by the auth middleware; routes without it (e.g. /init) count as logged out.
	session, hasSession := models.SessionFromContext(r.Context())
	loggedIn := hasSession

	isStaff := false
	isAdmin := false

	if hasSession && session.Role == models.SESSION_ROLE_STAFF {
		isStaff = false
		staff, err := models.GetStaffFromUsername(session.User)
		if err != nil { w.WriteHeader(InternalServerError); return }
		if staff.Username != "" { isStaff = true }

		isAdmin = false
		if staff.Admin { isAdmin = true }
	}

	dataAny := r.Context().Value(_MAIN_DATA)
	var data *_MainData
	if dataAny != nil {
		data = dataAny.(*_MainData)
	}
	if data == nil {
		data = &_MainData{}
	}
	data.IsStaff = isStaff
	if hasSession { data.Username = session.User }
	data.IsAdmin = isAdmin
	data.LoggedIn = loggedIn
	data.IsUserApp = _SERVER_NAME == "User"

	errAny := r.Context().Value(_ERR)
	var errStr string
	if errAny != nil { errStr = errAny.(error).Error() }

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	statusAny := r.Context().Value(_STATUS)
	if statusAny != nil { w.WriteHeader(statusAny.(int)) }

	err := _Tpl.Execute(w, _TplData{
		Title: titleCaser.String(section),
		Server: _SERVER_NAME,
		Init: init, 
		MainName: "main-"+section,
		MainData: data,
		Err: errStr,
	})
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), InternalServerError)
		return
	}
}

// ctxUsername is the logged-in username set by the auth middleware ("" if none).
func ctxUsername(r *http.Request) string {
	s, _ := models.SessionFromContext(r.Context())
	return s.User
}

func serveResponse(
  w      http.ResponseWriter,
  r      *http.Request,
  data   *_MainData,
  status int,
  err    error,
) {
  ctx := context.WithValue(r.Context(), _MAIN_DATA, data)
  ctx =  context.WithValue(ctx, _STATUS, status)
  ctx =  context.WithValue(ctx, _ERR, err)
  ServeMainTemplate(w, r.WithContext(ctx))
}

func serveMsg(w http.ResponseWriter, r *http.Request, msg string) {
	serveResponse(w, r, &_MainData{Msg: msg}, OK, nil)
}

func serveData(w http.ResponseWriter, r *http.Request, data any) {
	serveResponse(w, r, &_MainData{Data: data}, OK, nil)
}

func serveNoResults(w http.ResponseWriter, r *http.Request) {
	serveMsg(w, r, _MsgNoResults)
}

func serveErr(w http.ResponseWriter, r *http.Request, status int, err error) {
	serveResponse(w, r, nil, status, err)
}

func serveBadRequest(w http.ResponseWriter, r *http.Request, err error) {
	serveErr(w, r, BadRequest, err)
}

func serveInternalErr(w http.ResponseWriter, r *http.Request) {
	serveErr(w, r, InternalServerError, _ErrInternal)
}

func serveResponseHX(
	w 					http.ResponseWriter,
	msg         string,
	status      int,
	nextAction 	*_NextAction,
) {
	serveOOB(w, status, _HXData{Msg: msg, NextAction: nextAction})
}

// _Frag is an extra template rendered after the response, usually an OOB swap.
type _Frag struct {
	Name string
	Data any
}

func frag(name string, data any) _Frag { return _Frag{name, data} }

// serveOOB writes the div-response message, its toast, then each fragment.
func serveOOB(w http.ResponseWriter, status int, data _HXData, frags ..._Frag) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	for _, f := range append([]_Frag{{"div-response", data}, {"toast", data.Msg}}, frags...) {
		if err := _Tpl.ExecuteTemplate(w, f.Name, f.Data); err != nil { slog.Error(err.Error()) }
	}
}

func serveBadRequestHX(w http.ResponseWriter, msg string) {
	serveResponseHX(w, msg, BadRequest, nil)
}

func serveInternalErrHX(w http.ResponseWriter) {
	serveResponseHX(w, _ErrInternal.Error(), InternalServerError, nil)
}

// isHX reports whether the request came from htmx (or a fetch that sets the header).
func isHX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// serveErrAuto answers with the div-response fragment for htmx requests and
// the full error page otherwise.
func serveErrAuto(w http.ResponseWriter, r *http.Request, status int, err error) {
	if isHX(r) { serveResponseHX(w, err.Error(), status, nil); return }
	serveErr(w, r, status, err)
}

func serveInternalErrAuto(w http.ResponseWriter, r *http.Request) {
	if isHX(r) { serveInternalErrHX(w); return }
	serveInternalErr(w, r)
}

// serveDataHX renders tplName directly (a bare partial, no page chrome) when
// the request came from htmx, and falls back to the normal full-page
// serveData otherwise.
func serveDataHX(w http.ResponseWriter, r *http.Request, data any, tplName string) {
	if isHX(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := _Tpl.ExecuteTemplate(w, tplName, data); err != nil {
			slog.Error(err.Error())
			serveInternalErrHX(w)
		}
		return
	}
	serveData(w, r, data)
}
