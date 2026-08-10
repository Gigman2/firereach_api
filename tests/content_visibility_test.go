//go:build integration

package tests

import (
	"context"
	"testing"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/infra/repo"
)

// N2: the public /v1/content endpoint is unauthenticated, so excluding draft
// content must happen at the repository, not only in the app client that
// happens to consume this API today.
//
// withdrawn must NOT be excluded here, even though it must never be
// *displayed*. Withdrawal is a signal the client needs, not content to hide
// server-side: the app's visibleItems()/effectiveState() already treat
// `withdrawn` as invisible to the user, and refreshContent unions the
// response with the bundled floor by slug. If List filtered withdrawn rows
// out, a withdrawn guide would simply be absent from the response, and that
// union would silently restore the bundled copy of the same slug in its
// place — undoing the withdrawal on every client. The client must receive
// the tombstone in order to honour it.
func TestListExcludesDraftButIncludesWithdrawn(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	contentRepo := repo.NewContentRepo(pool)

	slugs := map[string]string{
		"draft":          "vis-draft-probe",
		"pending_review": "vis-pending-probe",
		"withdrawn":      "vis-withdrawn-probe",
	}

	for state, slug := range slugs {
		state, slug := state, slug
		_, err := pool.Exec(ctx, `
			INSERT INTO safety_content (category, subcategory, slug, title, body, review_state)
			VALUES ('hazard', 'cooking', $1, 'T', 'B', $2)`,
			slug, state)
		if err != nil {
			t.Fatalf("seed %s row: %v", state, err)
		}
		t.Cleanup(func() {
			if _, err := pool.Exec(context.Background(), `DELETE FROM safety_content WHERE slug = $1`, slug); err != nil {
				t.Errorf("cleanup %s row: %v", state, err)
			}
		})
	}

	items, err := contentRepo.List(ctx, "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	bySlug := map[string]bool{}
	for _, item := range items {
		bySlug[item.Slug] = true
	}

	if bySlug[slugs["draft"]] {
		t.Errorf("List returned a draft item (%s); draft content must never be public", slugs["draft"])
	}
	if !bySlug[slugs["withdrawn"]] {
		t.Errorf("List did not return a withdrawn item (%s); the client needs the tombstone to honour the withdrawal itself", slugs["withdrawn"])
	}
	if !bySlug[slugs["pending_review"]] {
		t.Errorf("List did not return a pending_review item (%s); that state must remain public", slugs["pending_review"])
	}
}

// N2: GetByID must apply the same review-state filter as List. Before this
// fix, GetByID had no review_state filter in its WHERE clause at all — only
// a post-scan check that rejected `withdrawn` — so a direct-by-ID fetch let
// draft rows through on this unauthenticated endpoint, contradicting the
// invariant List enforces one function above. A withdrawn row, by contrast,
// must still be reachable by ID: it's the same tombstone-delivery reasoning
// as List above.
func TestGetByIDExcludesDraftButIncludesWithdrawn(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	contentRepo := repo.NewContentRepo(pool)

	ids := map[string]string{}
	for state, slug := range map[string]string{
		"draft":     "vis-draft-getbyid-probe",
		"withdrawn": "vis-withdrawn-getbyid-probe",
	} {
		state, slug := state, slug
		var id string
		err := pool.QueryRow(ctx, `
			INSERT INTO safety_content (category, subcategory, slug, title, body, review_state)
			VALUES ('hazard', 'cooking', $1, 'T', 'B', $2)
			RETURNING id`, slug, state).Scan(&id)
		if err != nil {
			t.Fatalf("seed %s row: %v", state, err)
		}
		ids[state] = id
		t.Cleanup(func() {
			if _, err := pool.Exec(context.Background(), `DELETE FROM safety_content WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup %s row: %v", state, err)
			}
		})
	}

	if _, err := contentRepo.GetByID(ctx, ids["draft"]); err != domain.ErrNotFound {
		t.Errorf("GetByID on a draft row = %v, want domain.ErrNotFound", err)
	}

	got, err := contentRepo.GetByID(ctx, ids["withdrawn"])
	if err != nil {
		t.Fatalf("GetByID on a withdrawn row: %v", err)
	}
	if got.Review.State != domain.ReviewStateWithdrawn {
		t.Errorf("GetByID on a withdrawn row returned state %q, want %q", got.Review.State, domain.ReviewStateWithdrawn)
	}
}
