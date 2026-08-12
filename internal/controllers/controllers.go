package controllers

import (
	"errors"
	tpl "html/template"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	
	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/static"
)

type _TplData struct {
	Title    string
	Server   string
	MainTpl  string
	Init     bool
	LoggedIn bool
}

var (
  _DataNoResults = "No results found"

	_ErrBadSearch = errors.New("Bad search criteria")

	_Tpl = tpl.Must(tpl.ParseFS(static.FS, static.HTMDIR+"/*"))
)

func HandleRootUser(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/menu", http.StatusPermanentRedirect)
}

func HandleRootStaff(w http.ResponseWriter, r *http.Request) {
	if !models.CheckInit() {
		http.Redirect(w, r, "/init", http.StatusPermanentRedirect)
	} else {
		http.Redirect(w, r, "/menu", http.StatusPermanentRedirect)
	}
}

func ServeHTMX(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "path")
	if r.Header.Get("HX-Request") != "true" || path == "" {
		http.NotFound(w, r); return
	}
	http.ServeFileFS(w, r, static.FS, static.HXDIR+"/"+path)
}

var _SectionsCheckSessionStaff = []string{ "init", "menu", "login" }

func ServeTemplateStaff(w http.ResponseWriter, r *http.Request) {
	section := strings.SplitN(r.URL.Path, "/", 2)[1]

	init := true
	if section == "init" {
		if models.CheckInit() {
			http.Redirect(w, r, "/menu", http.StatusMovedPermanently)
			return
		}
		init = false
	}

	loggedIn := true
	if slices.Contains(_SectionsCheckSessionStaff, section) {
		loggedIn = checkSession(r)
	}

	err := _Tpl.Execute(w, _TplData{
		Title: strings.Title(section), 
		Server: "Staff", 
		MainTpl: section, 
		Init: init, 
		LoggedIn: loggedIn,
	})
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
