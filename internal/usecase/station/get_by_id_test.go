package station_test

import (
	"context"
	"errors"
	"testing"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/mocks"
	"github.com/firereach/api/internal/usecase/station"
)

func TestGetStation_Success(t *testing.T) {
	repo := &mocks.StationRepo{
		GetByIDFunc: func(ctx context.Context, id string) (*domain.Station, error) {
			return &domain.Station{ID: "1", Name: "Test Station", Active: true}, nil
		},
	}

	uc := station.NewGetStation(repo)
	s, err := uc.Execute(context.Background(), "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "Test Station" {
		t.Errorf("expected 'Test Station', got '%s'", s.Name)
	}
}

func TestGetStation_NotFound(t *testing.T) {
	repo := &mocks.StationRepo{
		GetByIDFunc: func(ctx context.Context, id string) (*domain.Station, error) {
			return nil, domain.ErrNotFound
		},
	}

	uc := station.NewGetStation(repo)
	_, err := uc.Execute(context.Background(), "999")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetStation_InactiveStation(t *testing.T) {
	repo := &mocks.StationRepo{
		GetByIDFunc: func(ctx context.Context, id string) (*domain.Station, error) {
			return &domain.Station{ID: "1", Name: "Closed Station", Active: false}, nil
		},
	}

	uc := station.NewGetStation(repo)
	_, err := uc.Execute(context.Background(), "1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound for inactive station, got %v", err)
	}
}
