//go:build integration

package tests

import (
	"context"
	"strings"
	"testing"
)

// The database must refuse a row that claims review without provenance.
// This is the defect the whole feature exists to prevent, so it is pinned
// at the storage layer rather than trusted to application code.
func TestReviewedRowRequiresProvenance(t *testing.T) {
	pool := testPool(t) // existing helper in this package
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		INSERT INTO safety_content (category, subcategory, slug, title, body, review_state)
		VALUES ('first_aid', 'burns', 'test-no-provenance', 'T', 'B', 'reviewed')`)

	if err == nil {
		t.Fatal("expected the reviewed_requires_provenance constraint to reject this row")
	}
	if !strings.Contains(err.Error(), "reviewed_requires_provenance") {
		t.Fatalf("rejected for the wrong reason: %v", err)
	}
}

func TestInvalidReviewStateRejected(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		INSERT INTO safety_content (category, subcategory, slug, title, body, review_state)
		VALUES ('hazard', 'cooking', 'test-bad-state', 'T', 'B', 'approved')`)

	if err == nil {
		t.Fatal("expected review_state_valid to reject 'approved'")
	}
}
