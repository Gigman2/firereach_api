package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/firereach/api/internal/domain"
)

// tokenLifetime is how long a login stays valid.
const tokenLifetime = 24 * time.Hour

type Login struct {
	repo      domain.AdminRepository
	jwtSecret string
}

func NewLogin(r domain.AdminRepository, jwtSecret string) *Login {
	return &Login{repo: r, jwtSecret: jwtSecret}
}

// Execute returns a signed token, or an error wrapping ErrUnauthorized when
// the email is unknown or the password is wrong. Any other failure, such as
// the database being down, comes back as itself, never as bad credentials.
func (uc *Login) Execute(ctx context.Context, email, password string) (string, error) {
	a, err := uc.repo.GetByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		return "", fmt.Errorf("login: %w", domain.ErrUnauthorized)
	}
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password)); err != nil {
		return "", fmt.Errorf("login: %w", domain.ErrUnauthorized)
	}

	now := time.Now()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": a.ID,
		"exp": now.Add(tokenLifetime).Unix(),
		"iat": now.Unix(),
	}).SignedString([]byte(uc.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("login: sign token: %w", err)
	}
	return token, nil
}
