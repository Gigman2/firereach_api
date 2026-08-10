package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/firereach/api/internal/domain"
)

func TestContentResponseCarriesReviewProvenance(t *testing.T) {
	when := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)

	resp := ToContentResponse(domain.SafetyContent{
		ID:          "uuid-1",
		Slug:        "firstaid-burns",
		Category:    "first_aid",
		Subcategory: "burns",
		Title:       "Burns",
		ContentHash: "abc123",
		Review: domain.Review{
			State:       domain.ReviewStateReviewed,
			Name:        "A. Mensah",
			Credential:  "Ghana Health Service",
			ReviewedAt:  &when,
			ContentHash: "abc123",
		},
	})

	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out["slug"] != "firstaid-burns" {
		t.Errorf("slug missing from wire shape: %v", out["slug"])
	}
	if out["content_hash"] != "abc123" {
		t.Errorf("content_hash missing: %v", out["content_hash"])
	}

	review, ok := out["review"].(map[string]any)
	if !ok {
		t.Fatal("review block missing from wire shape — the badge cannot be data-driven without it")
	}
	if review["state"] != "reviewed" {
		t.Errorf("review.state = %v, want reviewed", review["state"])
	}
	if review["reviewer_name"] != "A. Mensah" {
		t.Errorf("review.reviewer_name = %v", review["reviewer_name"])
	}
	if review["content_hash"] != "abc123" {
		t.Errorf("review.content_hash = %v", review["content_hash"])
	}
	if review["reviewed_at"] == nil {
		t.Error("review.reviewed_at missing")
	}
}

func TestUnreviewedItemOmitsReviewerFields(t *testing.T) {
	resp := ToContentResponse(domain.SafetyContent{
		Slug:  "hazard-cooking",
		Title: "Cooking",
		Review: domain.Review{
			State: domain.ReviewStatePendingReview,
		},
	})

	raw, _ := json.Marshal(resp)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)

	review := out["review"].(map[string]any)
	if review["state"] != "pending_review" {
		t.Errorf("review.state = %v", review["state"])
	}
	if _, present := review["reviewer_name"]; present {
		t.Error("reviewer_name must be omitted when there is no reviewer")
	}
}
