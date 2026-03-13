package station

import (
	"context"
	"fmt"

	"github.com/firereach/api/internal/domain"
)

type DeactivateStation struct {
	repo domain.StationRepository
}

func NewDeactivateStation(r domain.StationRepository) *DeactivateStation {
	return &DeactivateStation{repo: r}
}

func (uc *DeactivateStation) Execute(ctx context.Context, id string) error {
	s, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("deactivate station: %w", err)
	}

	s.Active = false
	if err := uc.repo.Update(ctx, *s); err != nil {
		return fmt.Errorf("deactivate station: %w", err)
	}
	return nil
}
