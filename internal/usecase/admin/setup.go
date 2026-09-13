package admin

import (
	"context"
	"fmt"

	"github.com/firereach/api/internal/domain"
)

type Setup struct {
	repo domain.AdminRepository
}

func NewSetup(r domain.AdminRepository) *Setup {
	return &Setup{repo: r}
}

// Completed reports whether any admin exists yet. It is the fast answer for
// the common case; Execute is what guarantees only one first admin.
func (uc *Setup) Completed(ctx context.Context) (bool, error) {
	n, err := uc.repo.Count(ctx)
	if err != nil {
		return false, fmt.Errorf("check setup: %w", err)
	}
	return n > 0, nil
}

// Execute creates the first admin. It returns an error wrapping
// ErrSetupCompleted when another setup got there first.
func (uc *Setup) Execute(ctx context.Context, email, password string) (*domain.AdminUser, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("setup: %w", err)
	}
	a, err := uc.repo.CreateFirst(ctx, email, hash)
	if err != nil {
		return nil, fmt.Errorf("setup: %w", err)
	}
	return a, nil
}
