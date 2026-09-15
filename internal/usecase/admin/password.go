package admin

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/firereach/api/internal/domain"
)

// ErrHashPassword means a password could not be hashed. Length is checked
// first, so what is left is a server fault rather than anything a caller sent.
var ErrHashPassword = errors.New("failed to hash password")

// maxPasswordBytes is bcrypt's own limit. It counts bytes, not characters, so a
// password written in a non-Latin script reaches it sooner than an ASCII one.
const maxPasswordBytes = 72

func hashPassword(password string) (string, error) {
	if len(password) > maxPasswordBytes {
		return "", fmt.Errorf("password must be at most 72 bytes: %w", domain.ErrInvalidInput)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrHashPassword, err)
	}
	return string(hash), nil
}
