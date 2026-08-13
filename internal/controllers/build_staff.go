//go:build STAFF

package controllers

import (
	"net/http"

	"github.com/ajderniz/repostele/internal/models"
)

const _SERVER_NAME = "Staff"

func checkInit(
	w http.ResponseWriter,
	r *http.Request,
	section string,
) (init, redirect bool) {
	init = true
	if section == "init" {
		if models.CheckInit() {
			http.Redirect(w, r, "/menu", http.StatusMovedPermanently)
			redirect = true
			return
		}
		init = false
	}
	return
}

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	if !models.CheckInit() {
		http.Redirect(w, r, "/init", http.StatusPermanentRedirect)
	} else {
		http.Redirect(w, r, "/menu", http.StatusPermanentRedirect)
	}
}
