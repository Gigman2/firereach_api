package station

import (
	"context"
	"fmt"

	"github.com/firereach/api/internal/domain"
)

type GetStation struct {
	repo domain.StationRepository
}

func NewGetStation(r domain.StationRepository) *GetStation {
	return &GetStation{repo: r}
}

func (uc *GetStation) Execute(ctx context.Context, id string) (*domain.Station, error) {
	s, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get station: %w", err)
	}
	if !s.Active {
		return nil, domain.ErrNotFound
	}
	return s, nil
}
