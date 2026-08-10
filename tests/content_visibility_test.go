//go:build integration

package tests

import (
	"context"
	"testing"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/infra/repo"
)

// M3: the public /v1/content endpoint is unauthenticated, so filtering
// withdrawn and draft content must happen at the repository, not only in
// the app client that happens to consume this API today. Withdrawal is the
// one mechanism for pulling content judged unsafe, and it must be enforced
// where the data lives.
func TestListExcludesDraftAndWithdrawn(t *testing.T) {
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
	if bySlug[slugs["withdrawn"]] {
		t.Errorf("List returned a withdrawn item (%s); withdrawal must be enforced server-side", slugs["withdrawn"])
	}
	if !bySlug[slugs["pending_review"]] {
		t.Errorf("List did not return a pending_review item (%s); that state must remain public", slugs["pending_review"])
	}
}

func TestGetByIDHidesWithdrawn(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	contentRepo := repo.NewContentRepo(pool)

	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO safety_content (category, subcategory, slug, title, body, review_state)
		VALUES ('hazard', 'cooking', 'vis-withdrawn-getbyid-probe', 'T', 'B', 'withdrawn')
		RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatalf("seed withdrawn row: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM safety_content WHERE id = $1`, id); err != nil {
			t.Errorf("cleanup withdrawn row: %v", err)
		}
	})

	_, err = contentRepo.GetByID(ctx, id)
	if err != domain.ErrNotFound {
		t.Fatalf("GetByID on a withdrawn row = %v, want domain.ErrNotFound", err)
	}
}
