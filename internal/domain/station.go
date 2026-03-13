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
}

type StationRepository interface {
	ListActive(ctx context.Context) ([]Station, error)
	GetByID(ctx context.Context, id string) (*Station, error)
	Create(ctx context.Context, s Station) error
	Update(ctx context.Context, s Station) error
}
