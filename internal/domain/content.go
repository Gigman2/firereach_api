package domain

import (
	"context"
	"time"
)

type ReviewState string

const (
	ReviewStateDraft         ReviewState = "draft"
	ReviewStatePendingReview ReviewState = "pending_review"
	ReviewStateReviewed      ReviewState = "reviewed"
	ReviewStateWithdrawn     ReviewState = "withdrawn"
)

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

// Review records who approved an item and what exactly they approved.
// ContentHash is the hash of the text at approval time; when it stops
// matching the item's current ContentHash, the review is stale.
type Review struct {
	State       ReviewState
	Name        string
	Credential  string
	ReviewedAt  *time.Time
	ContentHash string
}

type SafetyContent struct {
	ID                string
	Slug              string
	Category          string // hazard | first_aid
	Subcategory       string
	Title             string
	Summary           string
	Body              string
	Steps             []Step
	Tags              []string
	ContextualTrigger string
	Sources           []Source
	ContentHash       string
	Review            Review
}

type ContentRepository interface {
	List(ctx context.Context, category, subcategory string) ([]SafetyContent, error)
	GetByID(ctx context.Context, id string) (*SafetyContent, error)
}
