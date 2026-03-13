package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/firereach/api/internal/domain"
)

var _ domain.SubmissionRepository = (*SubmissionRepo)(nil)

type SubmissionRepo struct {
	pool *pgxpool.Pool
}

func NewSubmissionRepo(pool *pgxpool.Pool) *SubmissionRepo {
	return &SubmissionRepo{pool: pool}
}

func (r *SubmissionRepo) Create(ctx context.Context, s domain.Submission) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO submissions (station_id, type, suggested_value, note, device_hash, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, s.StationID, s.Type, s.SuggestedValue, s.Note, s.DeviceHash, s.Status)
	if err != nil {
		return fmt.Errorf("postgres create submission: %w", err)
	}
	return nil
}

func (r *SubmissionRepo) ListPending(ctx context.Context) ([]domain.Submission, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, station_id, type, suggested_value, note, device_hash, status, submitted_at, reviewed_at, admin_note
		FROM submissions
		WHERE status = 'pending'
		ORDER BY submitted_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("postgres list pending submissions: %w", err)
	}
	defer rows.Close()

	var subs []domain.Submission
	for rows.Next() {
		var s domain.Submission
		if err := rows.Scan(&s.ID, &s.StationID, &s.Type, &s.SuggestedValue, &s.Note, &s.DeviceHash, &s.Status, &s.SubmittedAt, &s.ReviewedAt, &s.AdminNote); err != nil {
			return nil, fmt.Errorf("postgres scan submission: %w", err)
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (r *SubmissionRepo) UpdateStatus(ctx context.Context, id string, status domain.SubmissionStatus, adminNote string) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE submissions
		SET status = $2, admin_note = $3, reviewed_at = now()
		WHERE id = $1
	`, id, status, adminNote)
	if err != nil {
		return fmt.Errorf("postgres update submission status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// GetByID is a helper used internally — not part of the domain interface.
func (r *SubmissionRepo) GetByID(ctx context.Context, id string) (*domain.Submission, error) {
	var s domain.Submission
	err := r.pool.QueryRow(ctx, `
		SELECT id, station_id, type, suggested_value, note, device_hash, status, submitted_at, reviewed_at, admin_note
		FROM submissions
		WHERE id = $1
	`, id).Scan(&s.ID, &s.StationID, &s.Type, &s.SuggestedValue, &s.Note, &s.DeviceHash, &s.Status, &s.SubmittedAt, &s.ReviewedAt, &s.AdminNote)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("postgres get submission: %w", err)
	}
	return &s, nil
}
