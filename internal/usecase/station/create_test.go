package station_test

import (
	"context"
	"errors"
	"testing"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/mocks"
	"github.com/firereach/api/internal/usecase/station"
)

func aStation() domain.Station {
	return domain.Station{
		Name:     "Takoradi Station",
		Region:   "Western",
		District: "Sekondi-Takoradi",
		Lat:      4.93,
		Lng:      -1.77,
		Contacts: []domain.StationContact{{Phone: "0312345678", Active: true}},
	}
}

func TestCreateStation_StoresAnActiveStationWithAnID(t *testing.T) {
	var saved domain.Station
	repo := &mocks.StationRepo{CreateFunc: func(ctx context.Context, s domain.Station) error {
		saved = s
		return nil
	}}

	created, err := station.NewCreateStation(repo).Execute(context.Background(), aStation())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !domain.ValidID(created.ID) || saved.ID != created.ID {
		t.Errorf("created id = %q, want a UUID stored with the station", created.ID)
	}
	if !saved.Active {
		t.Error("a new station should be active")
	}
}

// The prime meridian runs through Tema, so a station there has a longitude of
// exactly 0. It is a real coordinate, not a missing one.
func TestCreateStation_AcceptsZeroLongitude(t *testing.T) {
	repo := &mocks.StationRepo{CreateFunc: func(ctx context.Context, s domain.Station) error { return nil }}

	s := aStation()
	s.Name, s.Lat, s.Lng = "Tema Station", 5.67, 0
	if _, err := station.NewCreateStation(repo).Execute(context.Background(), s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateStation_RejectsCoordinatesOffTheEarth(t *testing.T) {
	tests := []struct {
		name     string
		lat, lng float64
	}{
		{"latitude above the north pole", 90.1, -1.77},
		{"latitude below the south pole", -90.1, -1.77},
		{"longitude past the date line", 4.93, 180.1},
		{"longitude before the date line", 4.93, -180.1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.StationRepo{CreateFunc: func(ctx context.Context, _ domain.Station) error {
				t.Error("an impossible coordinate reached the repository")
				return nil
			}}
			s := aStation()
			s.Lat, s.Lng = tt.lat, tt.lng
			_, err := station.NewCreateStation(repo).Execute(context.Background(), s)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}
