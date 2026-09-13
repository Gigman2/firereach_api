package admin

import (
	"context"
	"fmt"

	"github.com/firereach/api/internal/domain"
)

type CreateAdmin struct {
	repo domain.AdminRepository
}

func NewCreateAdmin(r domain.AdminRepository) *CreateAdmin {
	return &CreateAdmin{repo: r}
}

// Execute returns an error wrapping ErrAlreadyExists when the email is taken,
// and one wrapping ErrHashPassword when the password cannot be hashed.
func (uc *CreateAdmin) Execute(ctx context.Context, email, password string) (*domain.AdminUser, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("create admin: %w", err)
	}
	a, err := uc.repo.Create(ctx, email, hash)
	if err != nil {
		return nil, fmt.Errorf("create admin: %w", err)
	}
	return a, nil
}
