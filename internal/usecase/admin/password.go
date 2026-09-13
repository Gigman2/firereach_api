package admin

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// ErrHashPassword means a password could not be hashed. bcrypt refuses
// anything over 72 bytes, so a request can trigger it.
var ErrHashPassword = errors.New("failed to hash password")

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrHashPassword, err)
	}
	return string(hash), nil
}
