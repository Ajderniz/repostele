package bind

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ajderniz/repostele/pkg/errman"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
)

var _Validate = validator.New(validator.WithRequiredStructEnabled())
var _Decoder  = schema.NewDecoder()

var _DecodingErr = errors.New("Decoding error")
var _ValidationErr = errors.New("Validation error")

func JSON(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		errman.PrintError(err)
		return _DecodingErr
	}
	if err := _Validate.Struct(dst); err != nil {
		errman.PrintError(err)
		return _ValidationErr
	}
	return nil
}

func Form(r *http.Request, dst any) error {
	if err := r.ParseForm(); err != nil { 
		errman.PrintError(err)
		return errors.New("Parsing errror")
	}
	if err := _Decoder.Decode(dst, r.Form); err != nil {
		errman.PrintError(err)
		return _DecodingErr
	}
	if err := _Validate.Struct(dst); err != nil {
		errman.PrintError(err)
		return _ValidationErr
	}
	return nil
}

func FormValue(r *http.Request, dst any, key string, validate string) error {
	v := r.FormValue(key)
	if err := _Validate.Var(v, validate); err != nil {
		errman.PrintError(err)
		return _ValidationErr
	}
	dst = v
	return nil
}