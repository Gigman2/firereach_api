CREATE TYPE submission_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE IF NOT EXISTS submissions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    station_id      UUID REFERENCES stations(id),
    type            TEXT NOT NULL,
    suggested_value TEXT NOT NULL,
    note            TEXT,
    device_hash     TEXT NOT NULL,
    status          submission_status NOT NULL DEFAULT 'pending',
    submitted_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at     TIMESTAMPTZ,
    admin_note      TEXT
);
