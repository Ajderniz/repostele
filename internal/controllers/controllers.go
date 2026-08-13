package controllers

import (
	"bytes"
	"context"
	"errors"
	tpl "html/template"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/ajderniz/repostele/static"
)

const _MAIN_DATA = "main_data"
const _MAIN_ERR = "main_err"

type _MainData struct {
	Data any
	Msg  string
}

type _TplData struct {
	Title    string
	Server   string
	Init     bool
	LoggedIn bool
	MainName string
	MainData _MainData
	Err      string
}

var (
  _MsgNoResults = "No results found"

	_ErrBadSearch = errors.New("Bad search criteria")

	_Tpl *tpl.Template
)

func callTemplate(name string, data any) (tpl.HTML, error) {
	buf := bytes.NewBuffer([]byte{})
	err := _Tpl.ExecuteTemplate(buf, name, data)
	return tpl.HTML(buf.String()), err
}

func InitTemplate() {
	_Tpl = tpl.New("base")
	_Tpl.Funcs(tpl.FuncMap{"CallTemplate": callTemplate})
	tpl.Must(_Tpl.ParseFS(static.FS, static.HTMDIR+"/*"))
}

func ServeHTMX(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "path")
	if r.Header.Get("HX-Request") != "true" || path == "" {
		http.NotFound(w, r); return
	}
	http.ServeFileFS(w, r, static.FS, static.HXDIR+"/"+path)
}

var _SectionsCheckSessionStaff = []string{ "init", "menu", "login" }

func ServeMainTemplate(w http.ResponseWriter, r *http.Request) {
	section := strings.SplitN(r.URL.Path, "/", 2)[1]

	init, redirect := checkInit(w, r, section)
	if redirect { return }

	loggedIn := true
	if slices.Contains(_SectionsCheckSessionStaff, section) {
		loggedIn = checkSession(r)
	}

	dataAny := r.Context().Value(_MAIN_DATA)
	var data _MainData
	if dataAny != nil { data = dataAny.(_MainData) }

	errAny := r.Context().Value(_MAIN_ERR)
	var errStr string
	if errAny != nil { errStr = errAny.(string) }

	err := _Tpl.Execute(w, _TplData{
		Title: strings.Title(section), 
		Server: _SERVER_NAME,
		Init: init, 
		LoggedIn: loggedIn,
		MainName: section+"-main",
		MainData: data,
		Err: errStr,
	})
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func serveMainTplWithData(
  w http.ResponseWriter,
  r *http.Request,
  data any,
) {
  ctx := context.WithValue(r.Context(), _MAIN_DATA, data)
  ServeMainTemplate(w, r.WithContext(ctx))
}
