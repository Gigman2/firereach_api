package dto

import (
	"time"

	"github.com/firereach/api/internal/domain"
)

type StepResponse struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type SourceResponse struct {
	Title      string `json:"title"`
	Publisher  string `json:"publisher"`
	Year       string `json:"year"`
	URL        string `json:"url"`
	Unverified bool   `json:"unverified,omitempty"`
}

// ReviewResponse is the provenance behind the badge the app renders. Without
// it on the wire the badge has no choice but to be a hardcoded literal, which
// is the defect this feature exists to remove.
type ReviewResponse struct {
	State        string     `json:"state"`
	ReviewerName string     `json:"reviewer_name,omitempty"`
	Credential   string     `json:"reviewer_credential,omitempty"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	ContentHash  string     `json:"content_hash,omitempty"`
}

type ContentResponse struct {
	ID                string           `json:"id"`
	Slug              string           `json:"slug"`
	Category          string           `json:"category"`
	Subcategory       string           `json:"subcategory"`
	Title             string           `json:"title"`
	Summary           string           `json:"summary,omitempty"`
	Body              string           `json:"body"`
	Steps             []StepResponse   `json:"steps,omitempty"`
	Tags              []string         `json:"tags,omitempty"`
	ContextualTrigger string           `json:"contextual_trigger,omitempty"`
	Sources           []SourceResponse `json:"sources"`
	ContentHash       string           `json:"content_hash"`
	Review            ReviewResponse   `json:"review"`
}

type AskAIRequest struct {
	Question string `json:"question" binding:"required"`
	Topic    string `json:"topic"`
}

type AIStepResponse struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body"`
}

// AskAIResponse is the structured answer the chat renders per kind. Answer is
// a server-computed plain-text flattening kept for backward compatibility,
// accessibility, and logging; new clients render the structured fields.
type AskAIResponse struct {
	Kind    string           `json:"kind"`
	Title   string           `json:"title,omitempty"`
	Body    string           `json:"body"`
	Items   []AIStepResponse `json:"items,omitempty"`
	Ordered bool             `json:"ordered,omitempty"`
	Warning string           `json:"warning,omitempty"`
	Answer  string           `json:"answer"`
}

func ToAskAIResponse(r domain.AIResponse) AskAIResponse {
	items := make([]AIStepResponse, len(r.Items))
	for i, it := range r.Items {
		items[i] = AIStepResponse{Title: it.Title, Body: it.Body}
	}
	return AskAIResponse{
		Kind:    string(r.Kind),
		Title:   r.Title,
		Body:    r.Body,
		Items:   items,
		Ordered: r.Ordered,
		Warning: r.Warning,
		Answer:  r.Flatten(),
	}
}

func ToContentResponse(c domain.SafetyContent) ContentResponse {
	steps := make([]StepResponse, len(c.Steps))
	for i, s := range c.Steps {
		steps[i] = StepResponse{Title: s.Title, Body: s.Body}
	}

	sources := make([]SourceResponse, len(c.Sources))
	for i, s := range c.Sources {
		sources[i] = SourceResponse{
			Title: s.Title, Publisher: s.Publisher,
			Year: s.Year, URL: s.URL, Unverified: s.Unverified,
		}
	}

	// Recomputed rather than trusted from storage. If a row's text is tampered
	// with while its hash columns are left alone, a passthrough would ship
	// content_hash == review.content_hash and the app would render a trust badge
	// over text no reviewer ever saw.
	hash := domain.ContentHash(c.Slug, c.Title, c.Summary, c.Body, c.Steps, c.Sources)

	return ContentResponse{
		ID:                c.ID,
		Slug:              c.Slug,
		Category:          c.Category,
		Subcategory:       c.Subcategory,
		Title:             c.Title,
		Summary:           c.Summary,
		Body:              c.Body,
		Steps:             steps,
		Tags:              c.Tags,
		ContextualTrigger: c.ContextualTrigger,
		Sources:           sources,
		ContentHash:       hash,
		Review: ReviewResponse{
			State:        string(c.Review.State),
			ReviewerName: c.Review.Name,
			Credential:   c.Review.Credential,
			ReviewedAt:   c.Review.ReviewedAt,
			ContentHash:  c.Review.ContentHash,
		},
	}
}

func ToContentListResponse(items []domain.SafetyContent) []ContentResponse {
	result := make([]ContentResponse, len(items))
	for i, c := range items {
		result[i] = ToContentResponse(c)
	}
	return result
}
