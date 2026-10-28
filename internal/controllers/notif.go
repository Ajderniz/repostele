package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ajderniz/repostele/internal/models"

	"github.com/ajderniz/repostele/internal/reqlog"
)

func ServeNotif(w http.ResponseWriter, r *http.Request) {
	if !isHX(r) { http.NotFound(w, r); return }

	username, role, loggedIn := checkSessionUser(r)
	prev := r.URL.Query().Get("token")
	silent := r.URL.Query().Get("silent") == "1"

	if !loggedIn { noNotif(w, ""); return }
	if role == models.SESSION_ROLE_STAFF {
		serveStaffNotif(w, r, prev, silent)
		return
	}
	serveUserNotif(w, username, prev, silent, r.URL.Query().Get("widget") == "1")
}

func serveStaffNotif(w http.ResponseWriter, r *http.Request, prev string, silent bool) {
	stats, err := models.GetPendingQueueStats()
	if err != nil { reqlog.Error(r, err); noNotif(w, prev); return }
	token := fmt.Sprintf("%d|%d|%d", stats.Count, stats.MaxID, stats.MaxUpdated)
	if token == prev { noNotif(w, token); return }

	prevMaxID := 0
	if parts := strings.Split(prev, "|"); len(parts) == 3 {
		prevMaxID, _ = strconv.Atoi(parts[1])
	}


	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Notif-Token", token)
	if prev != "" && !silent {
		msg := "Cola de órdenes actualizada"
		if stats.MaxID > prevMaxID {
			msg = "Nueva orden #" + strconv.Itoa(stats.MaxID)
		}
		w.Header().Set("HX-Trigger", "order-changed")
		_Tpl.ExecuteTemplate(w, "toast", msg)
	}
	_Tpl.ExecuteTemplate(w, "pending-badge", map[string]any{
		"Id": "nav-pending-badge", "Count": stats.Unreviewed, "OOB": true,
	})
}

// hasSlot: the page has #current-order, so send its OOB update.
func serveUserNotif(w http.ResponseWriter, username, prev string, silent, hasSlot bool) {
	order, err := models.GetLatestOrderFromUsername(username)
	token := ""
	active := err == nil && order.Pending()
	if err == nil && order.RefNum != "" {
		token = fmt.Sprintf("%d|%d|%s|%d", order.Id, order.Status, order.RefNum, order.Updated)
	}
	if token == prev { noNotif(w, token); return }

	var widget *models.Order
	if active { widget = &order }

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Notif-Token", token)
	if prev != "" && !silent {
		msg := fmt.Sprintf("Tu orden #%d: %s", order.Id, orderStatusName(order.Status))
		w.Header().Set("HX-Trigger", "order-changed")
		_Tpl.ExecuteTemplate(w, "toast", msg)
	}
	if hasSlot {
		_Tpl.ExecuteTemplate(w, "current-order", map[string]any{
			"Order": widget, "OOB": true,
		})
	}
}

func noNotif(w http.ResponseWriter, token string) {
	w.Header().Set("X-Notif-Token", token)
	w.WriteHeader(http.StatusNoContent)
}
