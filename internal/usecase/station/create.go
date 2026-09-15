package station

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/firereach/api/internal/domain"
)

type CreateStation struct {
	repo domain.StationRepository
}

func NewCreateStation(r domain.StationRepository) *CreateStation {
	return &CreateStation{repo: r}
}

func (uc *CreateStation) Execute(ctx context.Context, s domain.Station) (*domain.Station, error) {
	if s.Name == "" || s.Region == "" || s.District == "" {
		return nil, fmt.Errorf("create station: %w", domain.ErrInvalidInput)
	}
	if !validCoordinates(s.Lat, s.Lng) {
		return nil, fmt.Errorf("create station: coordinates are not on the earth: %w", domain.ErrInvalidInput)
	}
	if len(s.Contacts) == 0 {
		return nil, fmt.Errorf("create station: at least one contact required: %w", domain.ErrInvalidInput)
	}

	s.ID = uuid.NewString()
	s.Active = true

	if err := uc.repo.Create(ctx, s); err != nil {
		return nil, fmt.Errorf("create station: %w", err)
	}
	return &s, nil
}

// validCoordinates reports whether a point is on the earth. It checks the whole
// range rather than a Ghana-shaped box, so a station near a border, or a
// correction that lands just outside one, is never refused for that.
func validCoordinates(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}
