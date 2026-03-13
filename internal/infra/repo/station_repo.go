package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/firereach/api/internal/domain"
)

var _ domain.StationRepository = (*StationRepo)(nil)

type StationRepo struct {
	pool *pgxpool.Pool
}

func NewStationRepo(pool *pgxpool.Pool) *StationRepo {
	return &StationRepo{pool: pool}
}

func (r *StationRepo) ListActive(ctx context.Context) ([]domain.Station, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.name, s.region, s.district, s.lat, s.lng, s.active, s.created_at, s.updated_at
		FROM stations s
		WHERE s.active = true
		ORDER BY s.name
	`)
	if err != nil {
		return nil, fmt.Errorf("postgres list active stations: %w", err)
	}
	defer rows.Close()

	var stations []domain.Station
	for rows.Next() {
		var s domain.Station
		if err := rows.Scan(&s.ID, &s.Name, &s.Region, &s.District, &s.Lat, &s.Lng, &s.Active, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("postgres scan station: %w", err)
		}
		stations = append(stations, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return r.attachContacts(ctx, stations)
}

func (r *StationRepo) GetByID(ctx context.Context, id string) (*domain.Station, error) {
	var s domain.Station
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, region, district, lat, lng, active, created_at, updated_at
		FROM stations
		WHERE id = $1
	`, id).Scan(&s.ID, &s.Name, &s.Region, &s.District, &s.Lat, &s.Lng, &s.Active, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("postgres get station: %w", err)
	}

	contacts, err := r.getContacts(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	s.Contacts = contacts
	return &s, nil
}

func (r *StationRepo) Create(ctx context.Context, s domain.Station) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO stations (id, name, region, district, lat, lng, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, s.ID, s.Name, s.Region, s.District, s.Lat, s.Lng, s.Active)
	if err != nil {
		return fmt.Errorf("postgres create station: %w", err)
	}

	for _, c := range s.Contacts {
		_, err = tx.Exec(ctx, `
			INSERT INTO station_contacts (station_id, phone, response_rate, active)
			VALUES ($1, $2, $3, $4)
		`, s.ID, c.Phone, c.ResponseRate, c.Active)
		if err != nil {
			return fmt.Errorf("postgres create station contact: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *StationRepo) Update(ctx context.Context, s domain.Station) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	ct, err := tx.Exec(ctx, `
		UPDATE stations
		SET name = $2, region = $3, district = $4, lat = $5, lng = $6, active = $7, updated_at = now()
		WHERE id = $1
	`, s.ID, s.Name, s.Region, s.District, s.Lat, s.Lng, s.Active)
	if err != nil {
		return fmt.Errorf("postgres update station: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	// Replace all contacts
	_, err = tx.Exec(ctx, `DELETE FROM station_contacts WHERE station_id = $1`, s.ID)
	if err != nil {
		return fmt.Errorf("postgres delete station contacts: %w", err)
	}

	for _, c := range s.Contacts {
		_, err = tx.Exec(ctx, `
			INSERT INTO station_contacts (station_id, phone, response_rate, active)
			VALUES ($1, $2, $3, $4)
		`, s.ID, c.Phone, c.ResponseRate, c.Active)
		if err != nil {
			return fmt.Errorf("postgres create station contact: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// attachContacts loads contacts for a batch of stations in a single query.
func (r *StationRepo) attachContacts(ctx context.Context, stations []domain.Station) ([]domain.Station, error) {
	if len(stations) == 0 {
		return stations, nil
	}

	ids := make([]string, len(stations))
	indexByID := make(map[string]int, len(stations))
	for i, s := range stations {
		ids[i] = s.ID
		indexByID[s.ID] = i
		stations[i].Contacts = []domain.StationContact{}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT station_id, phone, response_rate, active
		FROM station_contacts
		WHERE station_id = ANY($1)
		ORDER BY active DESC, response_rate DESC
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("postgres list station contacts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var stationID string
		var c domain.StationContact
		if err := rows.Scan(&stationID, &c.Phone, &c.ResponseRate, &c.Active); err != nil {
			return nil, fmt.Errorf("postgres scan station contact: %w", err)
		}
		if idx, ok := indexByID[stationID]; ok {
			stations[idx].Contacts = append(stations[idx].Contacts, c)
		}
	}

	return stations, rows.Err()
}

// getContacts loads contacts for a single station.
func (r *StationRepo) getContacts(ctx context.Context, stationID string) ([]domain.StationContact, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT phone, response_rate, active
		FROM station_contacts
		WHERE station_id = $1
		ORDER BY active DESC, response_rate DESC
	`, stationID)
	if err != nil {
		return nil, fmt.Errorf("postgres get station contacts: %w", err)
	}
	defer rows.Close()

	var contacts []domain.StationContact
	for rows.Next() {
		var c domain.StationContact
		if err := rows.Scan(&c.Phone, &c.ResponseRate, &c.Active); err != nil {
			return nil, fmt.Errorf("postgres scan station contact: %w", err)
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}
