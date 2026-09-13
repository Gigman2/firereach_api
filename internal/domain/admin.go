package domain

import (
	"context"
	"errors"
	"time"
)

type AdminUser struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// ErrSetupCompleted means the one-time first-admin setup has already run.
var ErrSetupCompleted = errors.New("setup already completed")

type AdminRepository interface {
	// GetByEmail returns ErrNotFound when no admin has that email.
	GetByEmail(ctx context.Context, email string) (*AdminUser, error)
	// Create returns ErrAlreadyExists when the email is already registered.
	Create(ctx context.Context, email, passwordHash string) (*AdminUser, error)
	List(ctx context.Context) ([]AdminUser, error)
	Count(ctx context.Context) (int, error)
	// CreateFirst creates the first admin atomically and returns
	// ErrSetupCompleted if any admin already exists, so two setups racing on
	// an empty table cannot both succeed.
	CreateFirst(ctx context.Context, email, passwordHash string) (*AdminUser, error)
}
