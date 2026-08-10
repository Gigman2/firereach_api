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

type Step struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type Source struct {
	Title      string `json:"title"`
	Publisher  string `json:"publisher"`
	Year       string `json:"year"`
	URL        string `json:"url"`
	Unverified bool   `json:"unverified,omitempty"`
}
