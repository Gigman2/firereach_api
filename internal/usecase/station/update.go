package station

import (
	"context"
	"fmt"

	"github.com/firereach/api/internal/domain"
)

type UpdateStation struct {
	repo domain.StationRepository
}

func NewUpdateStation(r domain.StationRepository) *UpdateStation {
	return &UpdateStation{repo: r}
}

func (uc *UpdateStation) Execute(ctx context.Context, s domain.Station) error {
	if s.ID == "" {
		return fmt.Errorf("update station: %w", domain.ErrInvalidInput)
	}

	existing, err := uc.repo.GetByID(ctx, s.ID)
	if err != nil {
		return fmt.Errorf("update station: %w", err)
	}

	// Merge — only overwrite fields that were provided
	if s.Name != "" {
		existing.Name = s.Name
	}
	if s.Region != "" {
		existing.Region = s.Region
	}
	if s.District != "" {
		existing.District = s.District
	}
	if s.Lat != 0 {
		existing.Lat = s.Lat
	}
	if s.Lng != 0 {
		existing.Lng = s.Lng
	}
	if s.Contacts != nil {
		existing.Contacts = s.Contacts
	}

	if err := uc.repo.Update(ctx, *existing); err != nil {
		return fmt.Errorf("update station: %w", err)
	}
	return nil
}
