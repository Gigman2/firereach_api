package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/firereach/api/internal/domain"
)

func TestContentResponseCarriesReviewProvenance(t *testing.T) {
	when := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)

	item := domain.SafetyContent{
		ID:          "uuid-1",
		Slug:        "firstaid-burns",
		Category:    "first_aid",
		Subcategory: "burns",
		Title:       "Burns",
	}
	realHash := domain.ContentHash(item.Slug, item.Title, item.Summary, item.Body, item.Steps, item.Sources)
	item.ContentHash = realHash
	item.Review = domain.Review{
		State:       domain.ReviewStateReviewed,
		Name:        "A. Mensah",
		Credential:  "Ghana Health Service",
		ReviewedAt:  &when,
		ContentHash: realHash,
	}

	resp := ToContentResponse(item)

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
	if out["content_hash"] != realHash {
		t.Errorf("content_hash = %v, want %v", out["content_hash"], realHash)
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
	if review["content_hash"] != realHash {
		t.Errorf("review.content_hash = %v, want %v", review["content_hash"], realHash)
	}
	if review["reviewed_at"] == nil {
		t.Error("review.reviewed_at missing")
	}
}

// TestContentHashIsRecomputedNotTrusted pins the security property directly:
// content_hash on the wire must always be the hash of the wire's own text,
// never a value trusted from storage. A stored ContentHash can go stale (a
// bad seed, a direct SQL edit, a compromised OTA source can change Body
// while leaving the hash columns untouched); if ToContentResponse ever
// passes c.ContentHash through instead of recomputing it, a tampered row
// would ship content_hash == review.content_hash and the app would render
// a trust badge over text no reviewer ever approved. Do not "simplify" this
// back to passthrough.
func TestContentHashIsRecomputedNotTrusted(t *testing.T) {
	const staleZeroHash = "0000000000000000000000000000000000000000000000000000000000000000"

	item := domain.SafetyContent{
		Slug:        "firstaid-burns",
		Title:       "Burns",
		Body:        "TAMPERED: text nobody reviewed",
		ContentHash: staleZeroHash,
		Review: domain.Review{
			State:       domain.ReviewStateReviewed,
			Name:        "A. Mensah",
			ContentHash: staleZeroHash,
		},
	}

	resp := ToContentResponse(item)

	realHash := domain.ContentHash(item.Slug, item.Title, item.Summary, item.Body, item.Steps, item.Sources)

	if resp.ContentHash == staleZeroHash {
		t.Fatal("content_hash was trusted from storage instead of recomputed — a tampered row would carry a stale-but-matching hash")
	}
	if resp.ContentHash != realHash {
		t.Errorf("content_hash = %v, want recomputed digest %v", resp.ContentHash, realHash)
	}
	if resp.ContentHash == resp.Review.ContentHash {
		t.Error("content_hash must not equal review.content_hash here — the whole point is that they diverge when text is tampered with")
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
