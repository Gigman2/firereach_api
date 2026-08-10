package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/firereach/api/internal/domain"
)

var _ domain.ContentRepository = (*ContentRepo)(nil)

type ContentRepo struct {
	pool *pgxpool.Pool
}

func NewContentRepo(pool *pgxpool.Pool) *ContentRepo {
	return &ContentRepo{pool: pool}
}

const contentColumns = `
    id, slug, category, subcategory, title, summary, body,
    steps, tags, contextual_trigger, sources, content_hash,
    review_state, reviewer_name, reviewer_credential, last_reviewed, reviewed_content_hash`

// rowScanner is satisfied by both pgx.Row and pgx.Rows, so List and GetByID
// share one definition of how a safety_content row maps to the domain.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanContent(row rowScanner) (domain.SafetyContent, error) {
	var (
		c            domain.SafetyContent
		stepsRaw     []byte
		tagsRaw      []byte
		sourcesRaw   []byte
		trigger      *string
		reviewerName *string
		credential   *string
		reviewedAt   *time.Time
		reviewedHash *string
		state        string
	)

	err := row.Scan(
		&c.ID, &c.Slug, &c.Category, &c.Subcategory, &c.Title, &c.Summary, &c.Body,
		&stepsRaw, &tagsRaw, &trigger, &sourcesRaw, &c.ContentHash,
		&state, &reviewerName, &credential, &reviewedAt, &reviewedHash,
	)
	if err != nil {
		return domain.SafetyContent{}, err
	}

	if len(stepsRaw) > 0 {
		if err := json.Unmarshal(stepsRaw, &c.Steps); err != nil {
			return domain.SafetyContent{}, fmt.Errorf("decode steps for %s: %w", c.Slug, err)
		}
	}
	if len(tagsRaw) > 0 {
		if err := json.Unmarshal(tagsRaw, &c.Tags); err != nil {
			return domain.SafetyContent{}, fmt.Errorf("decode tags for %s: %w", c.Slug, err)
		}
	}
	if len(sourcesRaw) > 0 {
		if err := json.Unmarshal(sourcesRaw, &c.Sources); err != nil {
			return domain.SafetyContent{}, fmt.Errorf("decode sources for %s: %w", c.Slug, err)
		}
	}

	deref := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}

	c.ContextualTrigger = deref(trigger)
	c.Review = domain.Review{
		State:       domain.ReviewState(state),
		Name:        deref(reviewerName),
		Credential:  deref(credential),
		ReviewedAt:  reviewedAt,
		ContentHash: deref(reviewedHash),
	}

	return c, nil
}

// List and GetByID both filter to what is safe for any unauthenticated
// client to see. The public API has no auth on this endpoint, and
// withdrawal is the one mechanism for pulling content judged unsafe — it
// must be enforced where the data lives, not only by the app client. draft
// rows are excluded too: they are work in progress, not yet offered for
// review, and were never meant to be public.
const publicReviewStates = `review_state IN ('pending_review', 'reviewed')`

func (r *ContentRepo) List(ctx context.Context, category, subcategory string) ([]domain.SafetyContent, error) {
	query := `
		SELECT ` + contentColumns + `
		FROM safety_content
		WHERE ($1 = '' OR category = $1)
		  AND ($2 = '' OR subcategory = $2)
		  AND ` + publicReviewStates + `
		ORDER BY category, subcategory, title
	`
	rows, err := r.pool.Query(ctx, query, category, subcategory)
	if err != nil {
		return nil, fmt.Errorf("postgres list content: %w", err)
	}
	defer rows.Close()

	var items []domain.SafetyContent
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres scan content: %w", err)
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (r *ContentRepo) GetByID(ctx context.Context, id string) (*domain.SafetyContent, error) {
	query := `
		SELECT ` + contentColumns + `
		FROM safety_content
		WHERE id = $1
	`
	c, err := scanContent(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("postgres get content: %w", err)
	}

	// Withdrawal is the one mechanism for pulling content judged unsafe; a
	// direct-by-ID fetch must honour it exactly like List does, or the
	// public endpoint returns a withdrawn item verbatim to anything that
	// isn't this app's own client-side filter.
	if c.Review.State == domain.ReviewStateWithdrawn {
		return nil, domain.ErrNotFound
	}

	return &c, nil
}
