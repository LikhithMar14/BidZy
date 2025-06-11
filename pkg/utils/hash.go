package utils


import (
	"encoding/base64"
	"math/rand"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string)(string, error) {
	bytes , err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes) , err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateOAuthState(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
