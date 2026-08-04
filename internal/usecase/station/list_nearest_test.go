package station_test

import (
	"context"
	"errors"
	"testing"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/mocks"
	"github.com/firereach/api/internal/usecase/station"
)

func TestListNearestStations_Success(t *testing.T) {
	repo := &mocks.StationRepo{
		ListActiveFunc: func(ctx context.Context) ([]domain.Station, error) {
			return []domain.Station{
				{ID: "1", Name: "Far Station", Lat: 6.0, Lng: -1.0, Active: true},
				{ID: "2", Name: "Near Station", Lat: 5.6, Lng: -0.2, Active: true},
				{ID: "3", Name: "Medium Station", Lat: 5.7, Lng: -0.5, Active: true},
			}, nil
		},
	}

	uc := station.NewListNearestStations(repo)
	// Query from Accra (5.6, -0.19)
	results, err := uc.Execute(context.Background(), 5.6, -0.19, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Name != "Near Station" {
		t.Errorf("expected nearest station first, got %s", results[0].Name)
	}
}

func TestListNearestStations_RepoError(t *testing.T) {
	repo := &mocks.StationRepo{
		ListActiveFunc: func(ctx context.Context) ([]domain.Station, error) {
			return nil, errors.New("db error")
		},
	}

	uc := station.NewListNearestStations(repo)
	_, err := uc.Execute(context.Background(), 5.6, -0.19, 3)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListNearestStations_EmptyList(t *testing.T) {
	repo := &mocks.StationRepo{
		ListActiveFunc: func(ctx context.Context) ([]domain.Station, error) {
			return []domain.Station{}, nil
		},
	}

	uc := station.NewListNearestStations(repo)
	results, err := uc.Execute(context.Background(), 5.6, -0.19, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestListNearestStations_PopulatesDistanceMeters(t *testing.T) {
	repo := &mocks.StationRepo{
		ListActiveFunc: func(ctx context.Context) ([]domain.Station, error) {
			return []domain.Station{
				{ID: "1", Name: "Exactly Here", Lat: 5.6, Lng: -0.19, Active: true},
				{ID: "2", Name: "Far Station", Lat: 6.0, Lng: -1.0, Active: true},
			}, nil
		},
	}

	uc := station.NewListNearestStations(repo)
	results, err := uc.Execute(context.Background(), 5.6, -0.19, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].DistanceMeters != 0 {
		t.Errorf("station at the query point should be 0m away, got %d", results[0].DistanceMeters)
	}
	if results[1].DistanceMeters <= 0 {
		t.Errorf("distant station should have positive distance, got %d", results[1].DistanceMeters)
	}
}

func TestListNearestStations_DistanceAscending(t *testing.T) {
	repo := &mocks.StationRepo{
		ListActiveFunc: func(ctx context.Context) ([]domain.Station, error) {
			return []domain.Station{
				{ID: "1", Name: "Far", Lat: 6.0, Lng: -1.0, Active: true},
				{ID: "2", Name: "Near", Lat: 5.6, Lng: -0.2, Active: true},
				{ID: "3", Name: "Medium", Lat: 5.7, Lng: -0.5, Active: true},
			}, nil
		},
	}

	uc := station.NewListNearestStations(repo)
	results, err := uc.Execute(context.Background(), 5.6, -0.19, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i < len(results); i++ {
		if results[i-1].DistanceMeters > results[i].DistanceMeters {
			t.Fatalf("results not ascending at index %d: %d > %d",
				i, results[i-1].DistanceMeters, results[i].DistanceMeters)
		}
	}
}

func TestListNearestStations_KnownDistanceIsPlausible(t *testing.T) {
	// 0.01 degrees of longitude at latitude 5.6 is roughly 1.108 km.
	repo := &mocks.StationRepo{
		ListActiveFunc: func(ctx context.Context) ([]domain.Station, error) {
			return []domain.Station{
				{ID: "1", Name: "One Notch East", Lat: 5.6, Lng: -0.18, Active: true},
			}, nil
		},
	}

	uc := station.NewListNearestStations(repo)
	results, err := uc.Execute(context.Background(), 5.6, -0.19, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := results[0].DistanceMeters
	if got < 1050 || got > 1160 {
		t.Errorf("expected roughly 1108m, got %d", got)
	}
}
