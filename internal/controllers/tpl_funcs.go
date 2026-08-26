package controllers

import (
	"bytes"
	tpl "html/template"
	"time"

	"github.com/ajderniz/repostele/internal/models"
)

func callTemplate(name string, data any) (tpl.HTML, error) {
	buf := bytes.NewBuffer([]byte{})
	err := _Tpl.ExecuteTemplate(buf, name, data)
	return tpl.HTML(buf.String()), err
}

func orderStatusName(s models.OrderStatus) string {
	switch s {
	case models.ORDER_STATUS_UNREVIEWED: return "Sin revisar"
	case models.ORDER_STATUS_DENIED:     return "Denegada"
	case models.ORDER_STATUS_CANCELLED:  return "Cancelada"
	case models.ORDER_STATUS_ACCEPTED:   return "Aceptada"
	case models.ORDER_STATUS_FULFILLED:  return "Cumplida"
	default: return ""
	}
}

func unixToTime(unix int64) string {
	t := time.Unix(unix, 0)
	return t.Format("2-1 03:04:05")
}

func dict(values ...any) map[string]any {
	d := make(map[string]any, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		key, _ := values[i].(string)
		d[key] = values[i+1]
	}
	return d
}

