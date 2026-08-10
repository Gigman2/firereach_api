//go:build integration

package tests

import (
	"context"
	"testing"
	"time"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/infra/repo"
)

// scanContent reads 17 columns into 17 domain.SafetyContent fields by
// position (contentColumns in content_repo.go must match the Scan() call
// order exactly). Nothing else in this suite pins that alignment: the other
// tests either exercise raw SQL directly or use content where a silent
// two-field swap — e.g. body landing in summary for a burns guide — could
// still parse and pass without anyone noticing. This test seeds a row with a
// distinct, recognizable marker in every text field and reads it back
// through the real ContentRepo.GetByID, so any column reordering makes a
// marker land in the wrong field and fails loudly.
func TestContentRoundTripFieldAlignment(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	contentRepo := repo.NewContentRepo(pool)

	reviewedAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO safety_content (
			slug, category, subcategory, title, summary, body,
			steps, tags, contextual_trigger, sources, content_hash,
			review_state, reviewer_name, reviewer_credential, last_reviewed, reviewed_content_hash
		) VALUES (
			'rt-probe', 'CATEGORY-MARKER', 'SUBCATEGORY-MARKER', 'TITLE-MARKER', 'SUMMARY-MARKER', 'BODY-MARKER',
			$1, $2, 'TRIGGER-MARKER', $3, 'CONTENT-HASH-MARKER',
			'reviewed', 'REVIEWER-NAME-MARKER', 'REVIEWER-CREDENTIAL-MARKER', $4, 'REVIEWED-HASH-MARKER'
		) RETURNING id`,
		`[{"title":"STEP-TITLE-MARKER","body":"STEP-BODY-MARKER"}]`,
		`["TAG-MARKER"]`,
		`[{"title":"SOURCE-TITLE-MARKER","publisher":"SOURCE-PUBLISHER-MARKER","year":"2024","url":"https://example.org/SOURCE-URL-MARKER"}]`,
		reviewedAt,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed probe row: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM safety_content WHERE id = $1`, id); err != nil {
			t.Errorf("cleanup probe row: %v", err)
		}
	})

	got, err := contentRepo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	stringChecks := []struct {
		field string
		got   string
		want  string
	}{
		{"Slug", got.Slug, "rt-probe"},
		{"Category", got.Category, "CATEGORY-MARKER"},
		{"Subcategory", got.Subcategory, "SUBCATEGORY-MARKER"},
		{"Title", got.Title, "TITLE-MARKER"},
		{"Summary", got.Summary, "SUMMARY-MARKER"},
		{"Body", got.Body, "BODY-MARKER"},
		{"ContextualTrigger", got.ContextualTrigger, "TRIGGER-MARKER"},
		{"ContentHash", got.ContentHash, "CONTENT-HASH-MARKER"},
		{"Review.Name", got.Review.Name, "REVIEWER-NAME-MARKER"},
		{"Review.Credential", got.Review.Credential, "REVIEWER-CREDENTIAL-MARKER"},
		{"Review.ContentHash", got.Review.ContentHash, "REVIEWED-HASH-MARKER"},
	}
	for _, c := range stringChecks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.field, c.got, c.want)
		}
	}

	if got.Review.State != domain.ReviewStateReviewed {
		t.Errorf("Review.State = %q, want %q", got.Review.State, domain.ReviewStateReviewed)
	}
	if len(got.Steps) != 1 || got.Steps[0].Title != "STEP-TITLE-MARKER" || got.Steps[0].Body != "STEP-BODY-MARKER" {
		t.Errorf("Steps = %+v, want a single STEP-TITLE-MARKER/STEP-BODY-MARKER step", got.Steps)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "TAG-MARKER" {
		t.Errorf("Tags = %+v, want [TAG-MARKER]", got.Tags)
	}
	if len(got.Sources) != 1 ||
		got.Sources[0].Title != "SOURCE-TITLE-MARKER" ||
		got.Sources[0].Publisher != "SOURCE-PUBLISHER-MARKER" ||
		got.Sources[0].URL != "https://example.org/SOURCE-URL-MARKER" {
		t.Errorf("Sources = %+v, want SOURCE-*-MARKER fields", got.Sources)
	}
	if got.Review.ReviewedAt == nil || !got.Review.ReviewedAt.Equal(reviewedAt) {
		t.Errorf("Review.ReviewedAt = %v, want %v", got.Review.ReviewedAt, reviewedAt)
	}
}
