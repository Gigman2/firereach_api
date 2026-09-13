package admin

import (
	"context"
	"fmt"

	"github.com/firereach/api/internal/domain"
)

type ListAdmins struct {
	repo domain.AdminRepository
}

func NewListAdmins(r domain.AdminRepository) *ListAdmins {
	return &ListAdmins{repo: r}
}

func (uc *ListAdmins) Execute(ctx context.Context) ([]domain.AdminUser, error) {
	admins, err := uc.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}
	return admins, nil
}
