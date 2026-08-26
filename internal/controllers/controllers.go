package controllers

import (
	"context"
	"errors"
	tpl "html/template"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/static"
)

const (
	_STATUS    = "ctx_status"
	_ERR       = "ctx_err"
	_MAIN_DATA = "main_data"

	OK = http.StatusOK
	Created = http.StatusCreated
	MovedPermanently = http.StatusMovedPermanently
	PermanentRedirect = http.StatusPermanentRedirect
	BadRequest = http.StatusBadRequest
	Unauthorized = http.StatusUnauthorized
	Forbidden = http.StatusForbidden
	Conflict = http.StatusConflict
	TooManyRequests = http.StatusTooManyRequests
	InternalServerError = http.StatusInternalServerError
)

type _MainData struct {
	Data     any
	Msg      string
	LoggedIn bool
	IsStaff  bool
	IsAdmin  bool
}

type _TplData struct {
	Title    string
	Server   string
	Init     bool
	LoggedIn bool
	IsStaff  bool
	IsAdmin  bool
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

func InitTemplate() {
	_Tpl = tpl.New("base")
	_Tpl.Funcs(tpl.FuncMap{
		"CallTemplate": callTemplate,
		"OrderStatusName": orderStatusName,
		"UnixToTime": unixToTime,
		"dict": dict,
	})
	tpl.Must(_Tpl.ParseFS(static.FS, static.HTMDIR+"/*"))
	tpl.Must(_Tpl.ParseFS(static.FS, static.HXDIR+"/*"))
}

func ServeHTMX(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "path")
	if r.Header.Get("HX-Request") != "true" || path == "" {
		http.NotFound(w, r); return
	}
	err := _Tpl.ExecuteTemplate(w, path, nil)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, _ErrInternal.Error(), InternalServerError);
	}
}

var _SectionsCheckSession = []string{ "init", "menu", "login" }

func ServeMainTemplate(w http.ResponseWriter, r *http.Request) {
	section := strings.Split(r.URL.Path, "/")[1]

	init, redirect := checkInit(w, r, section)
	if redirect { return }

	loggedIn := true
	if slices.Contains(_SectionsCheckSession, section) {
		loggedIn = checkSession(r)
	}

	isStaff := false
	isAdmin := false

	usernameAny := r.Context().Value(_CREDS_USERNAME)
	if usernameAny != nil {
		username := usernameAny.(string)

		isStaff = false
		staff, err := models.GetStaffFromUsername(username)
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
	data.IsAdmin = isAdmin
	data.LoggedIn = loggedIn

	errAny := r.Context().Value(_ERR)
	var errStr string
	if errAny != nil { errStr = errAny.(error).Error() }

	statusAny := r.Context().Value(_STATUS)
	if statusAny != nil { w.WriteHeader(statusAny.(int)) }

	err := _Tpl.Execute(w, _TplData{
		Title: titleCaser.String(section),
		Server: _SERVER_NAME,
		Init: init, 
		LoggedIn: loggedIn,
		IsStaff: isStaff,
		IsAdmin: isAdmin,
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
	data := _HXData {Msg: msg, NextAction: nextAction }
	w.WriteHeader(status)
	_Tpl.ExecuteTemplate(w, "div-response", data)
}

func serveBadRequestHX(w http.ResponseWriter, msg string) {
	serveResponseHX(w, msg, BadRequest, nil)
}

func serveInternalErrHX(w http.ResponseWriter) {
	serveResponseHX(w, _ErrInternal.Error(), InternalServerError, nil)
}

// serveDataHX renders tplName directly (a bare partial, no page chrome) when
// the request came from htmx, and falls back to the normal full-page
// serveData otherwise.
func serveDataHX(w http.ResponseWriter, r *http.Request, data any, tplName string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := _Tpl.ExecuteTemplate(w, tplName, data); err != nil {
			slog.Error(err.Error())
			serveInternalErrHX(w)
		}
		return
	}
	serveData(w, r, data)
}
