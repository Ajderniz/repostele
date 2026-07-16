package write

import (
	"encoding/json"
	"net/http"
)

type H map[string]any

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func ErrorJSON(w http.ResponseWriter, status int, err error) {
	JSON(w, status, H{"error": err.Error()})
}

func PlainText(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	w.Write([]byte(msg))
}