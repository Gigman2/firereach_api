package repo

import (
	"context"
	"encoding/json"
	"fmt"

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

func (r *ContentRepo) List(ctx context.Context, category, subcategory string) ([]domain.SafetyContent, error) {
	query := `
		SELECT id, category, subcategory, title, body, steps, tags, contextual_trigger, last_reviewed
		FROM safety_content
		WHERE ($1 = '' OR category = $1)
		  AND ($2 = '' OR subcategory = $2)
		ORDER BY category, subcategory, title
	`
	rows, err := r.pool.Query(ctx, query, category, subcategory)
	if err != nil {
		return nil, fmt.Errorf("postgres list content: %w", err)
	}
	defer rows.Close()

	var items []domain.SafetyContent
	for rows.Next() {
		var c domain.SafetyContent
		var stepsJSON, tagsJSON []byte
		if err := rows.Scan(&c.ID, &c.Category, &c.Subcategory, &c.Title, &c.Body, &stepsJSON, &tagsJSON, &c.ContextualTrigger, &c.LastReviewed); err != nil {
			return nil, fmt.Errorf("postgres scan content: %w", err)
		}
		_ = json.Unmarshal(stepsJSON, &c.Steps)
		_ = json.Unmarshal(tagsJSON, &c.Tags)
		items = append(items, c)
	}
	return items, rows.Err()
}

func (r *ContentRepo) GetByID(ctx context.Context, id string) (*domain.SafetyContent, error) {
	var c domain.SafetyContent
	var stepsJSON, tagsJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, category, subcategory, title, body, steps, tags, contextual_trigger, last_reviewed
		FROM safety_content
		WHERE id = $1
	`, id).Scan(&c.ID, &c.Category, &c.Subcategory, &c.Title, &c.Body, &stepsJSON, &tagsJSON, &c.ContextualTrigger, &c.LastReviewed)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("postgres get content: %w", err)
	}
	_ = json.Unmarshal(stepsJSON, &c.Steps)
	_ = json.Unmarshal(tagsJSON, &c.Tags)
	return &c, nil
}
