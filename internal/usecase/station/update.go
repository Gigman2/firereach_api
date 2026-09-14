package station

import (
	"context"
	"fmt"

	"github.com/firereach/api/internal/domain"
)

// StationPatch carries an admin's partial update: a field left empty or nil is
// kept as it is. Lat and Lng are pointers because 0 is a real coordinate. The
// prime meridian runs through Tema, and a float64 alone cannot tell a station
// there apart from one whose longitude was never sent.
type StationPatch struct {
	Name     string
	Region   string
	District string
	Lat      *float64
	Lng      *float64
	Contacts []domain.StationContact
}

type UpdateStation struct {
	repo domain.StationRepository
}

func NewUpdateStation(r domain.StationRepository) *UpdateStation {
	return &UpdateStation{repo: r}
}

func (uc *UpdateStation) Execute(ctx context.Context, id string, p StationPatch) error {
	if id == "" {
		return fmt.Errorf("update station: %w", domain.ErrInvalidInput)
	}

	existing, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("update station: %w", err)
	}

	if p.Name != "" {
		existing.Name = p.Name
	}
	if p.Region != "" {
		existing.Region = p.Region
	}
	if p.District != "" {
		existing.District = p.District
	}
	if p.Lat != nil {
		existing.Lat = *p.Lat
	}
	if p.Lng != nil {
		existing.Lng = *p.Lng
	}
	if p.Contacts != nil {
		existing.Contacts = p.Contacts
	}

	// Checked after the merge, so the test is the station that would be saved
	// rather than the half of it this request happened to carry.
	if !validCoordinates(existing.Lat, existing.Lng) {
		return fmt.Errorf("update station: coordinates are not on the earth: %w", domain.ErrInvalidInput)
	}

	if err := uc.repo.Update(ctx, *existing); err != nil {
		return fmt.Errorf("update station: %w", err)
	}
	return nil
}
