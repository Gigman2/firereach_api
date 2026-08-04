package domain

import (
	"context"
	"time"
)


type StationContact struct {
	Phone string
	ResponseRate float64
	Active       bool
}

type Station struct {
	ID        string
	Name      string
	Region    string
	District  string
	Lat       float64
	Lng       float64
	Contacts  []StationContact
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time

	// DistanceMeters is the great-circle distance from the coordinates the
	// caller supplied. It is computed per request and never persisted, so it
	// is zero on any read that had no reference point (e.g. GetByID).
	DistanceMeters int
}

type StationRepository interface {
	ListActive(ctx context.Context) ([]Station, error)
	GetByID(ctx context.Context, id string) (*Station, error)
	Create(ctx context.Context, s Station) error
	Update(ctx context.Context, s Station) error
}

// PrimaryPhone returns the active contact most likely to answer: the highest
// ResponseRate, ties broken by the lexicographically smallest phone so the
// result is stable regardless of row order. Returns "" when no contact is
// active, leaving the fallback decision to the caller.
func (s Station) PrimaryPhone() string {
	best := ""
	bestRate := -1.0
	for _, c := range s.Contacts {
		if !c.Active {
			continue
		}
		if c.ResponseRate > bestRate || (c.ResponseRate == bestRate && c.Phone < best) {
			best, bestRate = c.Phone, c.ResponseRate
		}
	}
	return best
}
