package pass

import (
	"crypto/rand"
	"encoding/base64"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/ajderniz/repostele/pkg/errman"
)

func HashPassword(pass string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
	if err != nil {
		errman.PrintError(err)
		return "", errors.New("Could not hash password")
	}
	return string(bytes), nil
}

func CheckPasswordHash(pass, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
	if err != nil {
		errman.PrintError(err)
		return errors.New("Wrong password")
	}
	return nil
}

func GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		errman.PrintError(err)
		return "", errors.New("Could not generate token")
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}