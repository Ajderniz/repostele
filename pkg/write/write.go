package write

import (
	"encoding/json"
	"net/http"
)

type H map[string]any

const (
	KEY_ERR = "error"
	KEY_MSG = "message"
	KEY_DAT = "data"
)

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func Error(w http.ResponseWriter, status int, err error) {
	JSON(w, status, H{KEY_ERR: err.Error()})
}

func Msg(w http.ResponseWriter, msg string) {
	JSON(w, http.StatusOK, H{KEY_MSG: msg})
}

func Data(w http.ResponseWriter, v any) {
	JSON(w, http.StatusOK, H{KEY_DAT: v})
}

func PlainText(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	w.Write([]byte(msg))
}