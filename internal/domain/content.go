package domain

import (
	"context"
	"time"
)

type SafetyContent struct {
	ID               string
	Category         string
	Subcategory      string
	Title            string
	Body             string
	Steps            []string
	Tags             []string
	ContextualTrigger string
	LastReviewed     time.Time
}

type ContentRepository interface {
	List(ctx context.Context, category, subcategory string) ([]SafetyContent, error)
	GetByID(ctx context.Context, id string) (*SafetyContent, error)
}
