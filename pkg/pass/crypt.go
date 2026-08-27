package pass

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(pass string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
	if err != nil {
		slog.Error(err.Error())
		return "", errors.New("No se pudo crear el hash de la contraseña")
	}
	return string(bytes), nil
}

func CheckPasswordHash(pass, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
	if err != nil {
		slog.Error(err.Error())
		return errors.New("Contraseña incorrecta")
	}
	return nil
}

func GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		slog.Error(err.Error())
		return "", errors.New("No se pudo generar el token")
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
