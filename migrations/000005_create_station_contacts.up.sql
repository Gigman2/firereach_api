CREATE TABLE IF NOT EXISTS station_contacts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    station_id    UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    phone         TEXT NOT NULL,
    response_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    active        BOOLEAN NOT NULL DEFAULT true
);

-- Migrate existing phone data into the new table
INSERT INTO station_contacts (station_id, phone, active)
SELECT id, phone, true FROM stations WHERE phone IS NOT NULL AND phone != '';

INSERT INTO station_contacts (station_id, phone, active)
SELECT id, phone_alt, true FROM stations WHERE phone_alt IS NOT NULL AND phone_alt != '';

-- Drop old columns
ALTER TABLE stations DROP COLUMN phone;
ALTER TABLE stations DROP COLUMN phone_alt;
