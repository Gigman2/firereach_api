package station_test

import (
	"context"
	"errors"
	"testing"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/mocks"
	"github.com/firereach/api/internal/usecase/station"
)

func coord(f float64) *float64 { return &f }

// existing is the station on file: Accra Central, west of the meridian.
func repoWithStation(t *testing.T, saved *domain.Station) *mocks.StationRepo {
	t.Helper()
	return &mocks.StationRepo{
		GetByIDFunc: func(ctx context.Context, id string) (*domain.Station, error) {
			return &domain.Station{
				ID: id, Name: "Accra Central", Region: "Greater Accra",
				District: "Accra Metropolitan", Lat: 5.55, Lng: -0.20, Active: true,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, s domain.Station) error {
			*saved = s
			return nil
		},
	}
}

// A longitude of 0 is Tema, not a missing value. The merge used to skip any
// zero, so a station could never be moved onto the prime meridian.
func TestUpdateStation_AppliesAZeroCoordinate(t *testing.T) {
	var saved domain.Station
	repo := repoWithStation(t, &saved)

	err := station.NewUpdateStation(repo).Execute(context.Background(), "5a0c5e1e-0000-4000-8000-000000000001",
		station.StationPatch{Lng: coord(0)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved.Lng != 0 {
		t.Errorf("lng = %v, want 0", saved.Lng)
	}
	if saved.Lat != 5.55 {
		t.Errorf("lat = %v, want the existing 5.55", saved.Lat)
	}
}

func TestUpdateStation_LeavesFieldsThatWereNotSent(t *testing.T) {
	var saved domain.Station
	repo := repoWithStation(t, &saved)

	err := station.NewUpdateStation(repo).Execute(context.Background(), "5a0c5e1e-0000-4000-8000-000000000001",
		station.StationPatch{Name: "Accra Central (Updated)"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved.Name != "Accra Central (Updated)" {
		t.Errorf("name = %q, want the new one", saved.Name)
	}
	if saved.Lat != 5.55 || saved.Lng != -0.20 {
		t.Errorf("coordinates = %v,%v, want the existing ones", saved.Lat, saved.Lng)
	}
}

func TestUpdateStation_RejectsCoordinatesOffTheEarth(t *testing.T) {
	repo := &mocks.StationRepo{
		GetByIDFunc: func(ctx context.Context, id string) (*domain.Station, error) {
			return &domain.Station{ID: id, Name: "Accra Central"}, nil
		},
		UpdateFunc: func(ctx context.Context, _ domain.Station) error {
			t.Error("an impossible coordinate reached the repository")
			return nil
		},
	}

	err := station.NewUpdateStation(repo).Execute(context.Background(), "5a0c5e1e-0000-4000-8000-000000000001",
		station.StationPatch{Lat: coord(91)})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateStation_MissingIDIsInvalidInput(t *testing.T) {
	repo := &mocks.StationRepo{}

	err := station.NewUpdateStation(repo).Execute(context.Background(), "", station.StationPatch{Name: "Ghost"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateStation_UnknownStationIsNotFound(t *testing.T) {
	repo := &mocks.StationRepo{
		GetByIDFunc: func(ctx context.Context, id string) (*domain.Station, error) {
			return nil, domain.ErrNotFound
		},
	}

	err := station.NewUpdateStation(repo).Execute(context.Background(), "5a0c5e1e-0000-4000-8000-0000000000ff",
		station.StationPatch{Name: "Ghost"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
