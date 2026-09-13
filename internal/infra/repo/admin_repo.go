package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/firereach/api/internal/domain"
)

// uniqueViolation is Postgres's error code for a broken UNIQUE constraint;
// on admin_users that can only be the email.
const uniqueViolation = "23505"

type AdminRepo struct {
	pool *pgxpool.Pool
}

func NewAdminRepo(pool *pgxpool.Pool) *AdminRepo {
	return &AdminRepo{pool: pool}
}

func (r *AdminRepo) GetByEmail(ctx context.Context, email string) (*domain.AdminUser, error) {
	var a domain.AdminUser
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at FROM admin_users WHERE email = $1`, email,
	).Scan(&a.ID, &a.Email, &a.PasswordHash, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get admin by email: %w", err)
	}
	return &a, nil
}

func (r *AdminRepo) Create(ctx context.Context, email, passwordHash string) (*domain.AdminUser, error) {
	return insertAdmin(ctx, r.pool, email, passwordHash)
}

func (r *AdminRepo) List(ctx context.Context) ([]domain.AdminUser, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, email, created_at FROM admin_users`)
	if err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}
	defer rows.Close()

	var admins []domain.AdminUser
	for rows.Next() {
		var a domain.AdminUser
		if err := rows.Scan(&a.ID, &a.Email, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan admin: %w", err)
		}
		admins = append(admins, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}
	return admins, nil
}

func (r *AdminRepo) Count(ctx context.Context) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count admins: %w", err)
	}
	return n, nil
}

// setupLockKey names first-admin setup for pg_advisory_xact_lock. Any value
// works as long as every setup uses the same one; this is ASCII "FREA".
const setupLockKey int64 = 0x46524541

// CreateFirst takes a transaction-scoped advisory lock before counting, so
// setups run one at a time: a second setup waits for the first to commit,
// then counts its admin and gets ErrSetupCompleted. Postgres releases the
// lock when the transaction ends, however it ends.
func (r *AdminRepo) CreateFirst(ctx context.Context, email, passwordHash string) (*domain.AdminUser, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin setup: %w", err)
	}
	defer tx.Rollback(ctx) // a no-op once committed

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, setupLockKey); err != nil {
		return nil, fmt.Errorf("lock setup: %w", err)
	}
	var n int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM admin_users`).Scan(&n); err != nil {
		return nil, fmt.Errorf("count admins: %w", err)
	}
	if n > 0 {
		return nil, domain.ErrSetupCompleted
	}
	a, err := insertAdmin(ctx, tx, email, passwordHash)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit setup: %w", err)
	}
	return a, nil
}

// querier is satisfied by both the pool and a transaction, so the insert is
// written once and shared.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func insertAdmin(ctx context.Context, q querier, email, passwordHash string) (*domain.AdminUser, error) {
	a := domain.AdminUser{Email: email, PasswordHash: passwordHash}
	err := q.QueryRow(ctx,
		`INSERT INTO admin_users (email, password_hash) VALUES ($1, $2) RETURNING id, created_at`,
		email, passwordHash,
	).Scan(&a.ID, &a.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return nil, domain.ErrAlreadyExists
	}
	if err != nil {
		return nil, fmt.Errorf("insert admin: %w", err)
	}
	return &a, nil
}
