ALTER TABLE stations ADD COLUMN phone TEXT NOT NULL DEFAULT '';
ALTER TABLE stations ADD COLUMN phone_alt TEXT;

-- Restore first contact as phone, second as phone_alt
UPDATE stations s SET phone = COALESCE(
    (SELECT sc.phone FROM station_contacts sc WHERE sc.station_id = s.id ORDER BY sc.active DESC LIMIT 1), ''
);

UPDATE stations s SET phone_alt = (
    SELECT sc.phone FROM station_contacts sc WHERE sc.station_id = s.id ORDER BY sc.active DESC OFFSET 1 LIMIT 1
);

DROP TABLE IF EXISTS station_contacts;
