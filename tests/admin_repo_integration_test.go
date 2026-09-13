//go:build integration

package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/infra/repo"
)

// emptyAdminPool gives each test its own throwaway database holding only the
// admin_users table, so the first-admin race can start from an empty table
// without touching a developer's real admins. The table is built from the
// real migration, so these tests run against the real schema and its UNIQUE
// email constraint.
func emptyAdminPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	server := testPool(t) // skips when DATABASE_URL is unset

	name := fmt.Sprintf("admin_repo_test_%d", time.Now().UnixNano())
	if _, err := server.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatalf("create throwaway database: %v", err)
	}

	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	cfg.ConnConfig.Database = name
	// Enough connections that every racer below holds its own.
	cfg.MaxConns = 60
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect to throwaway database: %v", err)
	}

	// Registered after testPool's cleanup, so it runs first: the throwaway
	// pool closes, then the still-open server pool drops the database.
	t.Cleanup(func() {
		pool.Close()
		drop := "DROP DATABASE IF EXISTS " + pgx.Identifier{name}.Sanitize() + " WITH (FORCE)"
		if _, err := server.Exec(context.Background(), drop); err != nil {
			t.Errorf("drop throwaway database %s: %v", name, err)
		}
	})

	migration, err := os.ReadFile("../migrations/000004_create_admin_users.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	return pool
}

func TestAdminRepo_CreateThenFindByEmail(t *testing.T) {
	r := repo.NewAdminRepo(emptyAdminPool(t))
	ctx := context.Background()

	created, err := r.Create(ctx, "a@firereach.test", "hash-a")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" || created.Email != "a@firereach.test" || created.CreatedAt.IsZero() {
		t.Fatalf("unexpected created admin: %+v", created)
	}

	found, err := r.GetByEmail(ctx, "a@firereach.test")
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if found.ID != created.ID || found.PasswordHash != "hash-a" {
		t.Fatalf("found %+v, want id %s with hash-a", found, created.ID)
	}
}

func TestAdminRepo_UnknownEmailIsNotFound(t *testing.T) {
	_, err := repo.NewAdminRepo(emptyAdminPool(t)).GetByEmail(context.Background(), "nobody@firereach.test")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestAdminRepo_DuplicateEmailIsAlreadyExists(t *testing.T) {
	r := repo.NewAdminRepo(emptyAdminPool(t))
	ctx := context.Background()
	if _, err := r.Create(ctx, "a@firereach.test", "hash-1"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := r.Create(ctx, "a@firereach.test", "hash-2"); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("second create err = %v, want ErrAlreadyExists", err)
	}
}

func TestAdminRepo_ListAndCount(t *testing.T) {
	r := repo.NewAdminRepo(emptyAdminPool(t))
	ctx := context.Background()
	for _, email := range []string{"a@firereach.test", "b@firereach.test"} {
		if _, err := r.Create(ctx, email, "hash"); err != nil {
			t.Fatalf("create %s: %v", email, err)
		}
	}
	list, err := r.List(ctx)
	if err != nil || len(list) != 2 {
		t.Fatalf("list = %d admins (err %v), want 2", len(list), err)
	}
	if n, err := r.Count(ctx); err != nil || n != 2 {
		t.Fatalf("count = %d (err %v), want 2", n, err)
	}
}

func TestAdminRepo_CreateFirstRefusesOnceAnAdminExists(t *testing.T) {
	r := repo.NewAdminRepo(emptyAdminPool(t))
	ctx := context.Background()
	if _, err := r.Create(ctx, "a@firereach.test", "hash"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := r.CreateFirst(ctx, "b@firereach.test", "hash"); !errors.Is(err, domain.ErrSetupCompleted) {
		t.Fatalf("err = %v, want ErrSetupCompleted", err)
	}
}

// The race this refactor closes: setup used to count admins and then insert
// in two separate steps, so simultaneous first-admin requests could all see
// an empty table and all succeed, through an endpoint that needs no login.
//
// Every racer starts with an already-open connection, all are released
// together, and the race runs several rounds. An earlier version of this test
// let each racer open its own connection and passed against the naive
// count-then-insert code: connection setup staggered the racers enough to
// hide the race, so it proved nothing.
func TestAdminRepo_CreateFirstLetsExactlyOneSimultaneousSetupWin(t *testing.T) {
	pool := emptyAdminPool(t)
	r := repo.NewAdminRepo(pool)
	ctx := context.Background()
	const racers, rounds = 50, 5

	// Open every connection up front, then hand them back to the pool idle.
	held := make([]*pgxpool.Conn, racers)
	for i := range held {
		c, err := pool.Acquire(ctx)
		if err != nil {
			t.Fatalf("open connection %d: %v", i, err)
		}
		held[i] = c
	}
	for _, c := range held {
		c.Release()
	}

	for round := 1; round <= rounds; round++ {
		if _, err := pool.Exec(ctx, "TRUNCATE admin_users"); err != nil {
			t.Fatalf("round %d: empty the table: %v", round, err)
		}

		var wg sync.WaitGroup
		start := make(chan struct{})
		errs := make([]error, racers)
		for i := 0; i < racers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				_, errs[i] = r.CreateFirst(ctx, fmt.Sprintf("racer%d@firereach.test", i), "hash")
			}(i)
		}
		close(start)
		wg.Wait()

		won := 0
		for i, err := range errs {
			switch {
			case err == nil:
				won++
			case errors.Is(err, domain.ErrSetupCompleted):
			default:
				t.Errorf("round %d: racer %d failed for the wrong reason: %v", round, i, err)
			}
		}
		n, err := r.Count(ctx)
		if err != nil || won != 1 || n != 1 {
			t.Fatalf("round %d: %d of %d simultaneous setups succeeded and %d admins exist (err %v), want exactly 1",
				round, won, racers, n, err)
		}
	}
}
