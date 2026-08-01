package controllers

import "errors"

var (
  _DataNoResults = "No results found"

	_ErrBadSearch = errors.New("Bad search criteria")
  _ErrGetAcc = errors.New("Could not retrieve account information")
)
