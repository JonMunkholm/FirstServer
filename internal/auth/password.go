package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword (password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		fmt.Printf("Error hashing password: %v", err)
		return "", err
	}


	return string(hashedPassword), nil
}


func CheckPasswordHash (password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
