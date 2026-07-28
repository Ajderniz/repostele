package controllers

import (
	"errors"
)

var (
	_ErrRegAcc = errors.New("Could not register account")
	_ErrSameUsername = errors.New("Username already exists")
	_ErrSamePassword = errors.New("Password is the same")

	_MsgAccNotFound = "Account not found"
  _MsgAccCreated = "Account registered successfully"
	_MsgLoggedOut = "Logged out successfully"
	_MsgAccDeactivated = "Account deactivated successfully"
	_MsgUsernameChanged = "Username changed successfully"
	_MsgPasswordChanged = "Passowrd changed successfully"
)
