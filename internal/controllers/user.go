package controllers

import (
	"errors"
	"net/http"
	"time"

	"github.com/ajderniz/repostele/internal/models"
	"github.com/ajderniz/repostele/pkg/bind"
	"github.com/ajderniz/repostele/pkg/errman"
	"github.com/ajderniz/repostele/pkg/pass"
	"github.com/ajderniz/repostele/pkg/write"
)

var _SessionTokenErr = errors.New("'session_token' cookie not found")

func logout(w http.ResponseWriter, r *http.Request) error {
  sessionToken, err := r.Cookie(models.SESSION_TOKEN)
  if err != nil {
    errman.PrintError(err)
    return _SessionTokenErr
  }
  if sessionToken.Value == "" {
    errman.PrintError(errors.New("session_token blank"))
    return _SessionTokenErr
  }
  return models.CloseSession(sessionToken.Value)
}

type _UserCreds struct {
  Username string `schema:"username" validate:"required,min=4,max=16,alphanum"`
  Password string `schema:"password" validate:"required,min=4,max=16,alphanum"`
}

func getUserAndCheckPassword(r *http.Request) (models.User, int, error) {
  creds := _UserCreds{}
  err := bind.Form(r, &creds)
  if err != nil { return models.User{}, http.StatusBadRequest, err }

  user, err := models.GetUserFromUsername(creds.Username)
  if err != nil { return models.User{}, http.StatusInternalServerError, err }
  if user.Username == "" { return models.User{}, http.StatusOK, nil }

  err = pass.CheckPasswordHash(creds.Password, user.PassHash)
  if err != nil { return models.User{}, http.StatusUnauthorized, err }

  return user, 0, nil
}

func RegisterUserAccount(w http.ResponseWriter, r *http.Request) {
  creds := _UserCreds{}
  err := bind.Form(r, &creds)
  if err != nil { write.ErrorJSON(w, http.StatusBadRequest, err); return }

  user := models.User{}
  user.Username = creds.Username
  user.PassHash, err = pass.HashPassword(creds.Password)
  if err != nil {write.ErrorJSON(w, http.StatusInternalServerError, err);return}
  user.TimeCreated = time.Now().Unix()

  err = models.InsertUser(user)
  if err != nil {write.ErrorJSON(w, http.StatusInternalServerError, err);return}

  write.JSON(w, http.StatusCreated, write.H{"message": "Account registered successfully"})
}

func UserLogin(w http.ResponseWriter, r *http.Request) {
  logout(w, r) // don't check for errors

  user, status, err := getUserAndCheckPassword(r)
  if err != nil { write.ErrorJSON(w, status, err); return }
  if user.Username == "" {
    write.JSON(w, http.StatusOK, write.H{"message": "User not found"})
    return
  }

  sessionToken, err := pass.GenerateToken(32)
  csrfToken,    err := pass.GenerateToken(32)
  if err != nil { write.ErrorJSON(w, http.StatusBadRequest, err); return }

  expires := time.Now().Add(24 * time.Hour)

  http.SetCookie(w, &http.Cookie{
    Name:     models.SESSION_TOKEN,
    Value:    sessionToken,
    Expires:  expires,
    HttpOnly: true,
  })
  http.SetCookie(w, &http.Cookie{
    Name:     models.SESSION_CSRF_TOKEN,
    Value:    csrfToken,
    Expires:  expires,
    HttpOnly: false,
  })

  err = models.OpenSession(models.Session{
    SessionToken: sessionToken,
    CSRFToken:    csrfToken,
    User:         user.Username,
    Starts:       time.Now().Unix(),
    Expires:      expires.Unix(),
  })
  if err != nil {write.ErrorJSON(w, http.StatusInternalServerError, err);return}

  write.JSON(w, http.StatusOK, write.H{
    "data": write.H{
      "username": user.Username,
      "time_created": user.TimeCreated,
    },
  })
}

func UserLogout(w http.ResponseWriter, r *http.Request) {
  err := logout(w, r)
  if err != nil { write.ErrorJSON(w, http.StatusBadRequest, err); return }

  http.SetCookie(w, &http.Cookie{
    Name:     models.SESSION_TOKEN,
    Value:    "",
    Expires:  time.Now().Add(-time.Hour),
    HttpOnly: true,
  })
  http.SetCookie(w, &http.Cookie{
    Name:     models.SESSION_CSRF_TOKEN,
    Value:    "",
    Expires:  time.Now().Add(-time.Hour),
    HttpOnly: false,
  })

  write.JSON(w, http.StatusOK, write.H{"message": "Logged out successfully"})
}

func DeactivateUserAccount(w http.ResponseWriter, r *http.Request) {
  username := r.Context().Value(models.USER_USERNAME).(string)

  latestOrder, err := models.GetLatestOrderFromUsername(username)
  if err != nil {write.ErrorJSON(w, http.StatusInternalServerError, err);return}
  if latestOrder.RefNum != "" &&
     latestOrder.Status != models.ORDER_STATUS_CANCELLED &&
     latestOrder.Status != models.ORDER_STATUS_FULFILLED {
    write.ErrorJSON(w, http.StatusConflict, errors.New("An order is pending."))
    return
  }

  err = logout(w, r)
  if err != nil {write.ErrorJSON(w, http.StatusInternalServerError, err);return}

  err = models.DeactivateUser(username)
  if err != nil {write.ErrorJSON(w, http.StatusInternalServerError, err);return}

  write.JSON(w, http.StatusOK, write.H{"message": "Account deleted successfully"})
}