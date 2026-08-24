package bind

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
)

var _Validate = validator.New(validator.WithRequiredStructEnabled())
var _Decoder  = schema.NewDecoder()

var _DecodingErr = errors.New("Decoding error")
var _ValidationErr = errors.New("Validation error")

func JSON(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		slog.Error(err.Error())
		return _DecodingErr
	}
	if err := _Validate.Struct(dst); err != nil {
		slog.Error(err.Error())
		return _ValidationErr
	}
	return nil
}

func Form(r *http.Request, dst any) error {
	if err := r.ParseForm(); err != nil { 
		slog.Error(err.Error())
		return errors.New("Parsing error")
	}
	if err := _Decoder.Decode(dst, r.Form); err != nil {
		slog.Error(err.Error())
		return _DecodingErr
	}
	if err := _Validate.Struct(dst); err != nil {
		slog.Error(err.Error())
		return _ValidationErr
	}
	return nil
}

func FormValue(r *http.Request, key, validate string) (string, error) {
	v := r.FormValue(key)
	if err := _Validate.VarWithKey(key, v, validate); err != nil {
		slog.Error(err.Error())
		return "", _ValidationErr
	}
	return v, nil
}

func URLParam(r *http.Request, key, validate string) (string, error) {
	v := chi.URLParam(r, key)
	if err := _Validate.VarWithKey(key, v, validate); err != nil {
		slog.Error(err.Error())
		return "", _ValidationErr
	}
	return v, nil
}
