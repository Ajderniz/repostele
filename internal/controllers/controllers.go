package controllers

import (
	"errors"
	tpl "html/template"
	"log/slog"
	"net/http"

	"github.com/ajderniz/repostele/internal/models"
	root "github.com/ajderniz/repostele/web"
)

type _TplData struct {
	Title  string
	Server string
}

const (
	HTMDIR  = root.HTMDIR
  HTMBASE = HTMDIR+"/base.html"
)

var (
  _DataNoResults = "No results found"

	_ErrBadSearch = errors.New("Bad search criteria")

	_TplInit *tpl.Template
)

func ParseTemplates() {
	_TplInit = tpl.Must(tpl.ParseFS(root.FS, HTMBASE))
	tpl.Must(_TplInit.ParseFS(root.FS, HTMDIR+"/init-main.html"))
}

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

func ServeInit(w http.ResponseWriter, r *http.Request) {
	err := _TplInit.Execute(w, _TplData{Title: "Init", Server: "Staff"})
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func ServeInitForm(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, root.FS, HTMDIR+"/staff-register-form.html")
}
