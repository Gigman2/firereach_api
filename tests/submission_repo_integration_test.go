//go:build integration

package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/infra/repo"
)

// submissionPool holds stations and submissions, which is what the foreign key
// between them needs.
func submissionPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return throwawayPool(t, "submission_repo_test",
		"000001_create_stations.up.sql",
		"000002_create_submissions.up.sql",
	)
}

func aSubmission(stationID *string) domain.Submission {
	return domain.Submission{
		StationID:      stationID,
		Type:           domain.SubmissionWrongPhone,
		SuggestedValue: "0302999999",
		DeviceHash:     "d_m4k2p1_8f3kq9z2x1",
		Status:         domain.SubmissionStatusPending,
	}
}

// A station id that is well formed but matches no row is the caller's mistake.
// Postgres answers with a foreign key violation, which the handler would
// otherwise report as a server fault.
func TestSubmissionRepo_UnknownStationIsInvalidInput(t *testing.T) {
	unknown := "5a0c5e1e-0000-4000-8000-0000000000ff"

	err := repo.NewSubmissionRepo(submissionPool(t)).Create(context.Background(), aSubmission(&unknown))
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
}

func TestSubmissionRepo_AcceptsAKnownStation(t *testing.T) {
	pool := submissionPool(t)
	ctx := context.Background()

	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO stations (name, region, district, lat, lng, phone)
		VALUES ('Tema Station', 'Greater Accra', 'Tema Metropolitan', 5.67, 0, '0303456789')
		RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatalf("insert station: %v", err)
	}

	if err := repo.NewSubmissionRepo(pool).Create(ctx, aSubmission(&id)); err != nil {
		t.Fatalf("create submission: %v", err)
	}
}

// A report about a station that is not on file at all carries no station id.
func TestSubmissionRepo_AcceptsNoStation(t *testing.T) {
	if err := repo.NewSubmissionRepo(submissionPool(t)).Create(context.Background(), aSubmission(nil)); err != nil {
		t.Fatalf("create submission: %v", err)
	}
}
