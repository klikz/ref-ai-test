-- Packing / line scan photos (ONVIF)
-- Apply:
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/108_packing_scan_photos.sql

BEGIN;

CREATE TABLE IF NOT EXISTS lines.packing_scan_photos (
    id          bigserial PRIMARY KEY,
    product_id  int REFERENCES lines.products(id),
    line_id     int NOT NULL DEFAULT 6,
    serial      varchar NOT NULL DEFAULT 'test',
    file_path   varchar NOT NULL,
    captured_at timestamptz NOT NULL DEFAULT now(),
    user_id     int
);

CREATE INDEX IF NOT EXISTS idx_packing_scan_photos_serial
    ON lines.packing_scan_photos (serial);

CREATE INDEX IF NOT EXISTS idx_packing_scan_photos_captured_at
    ON lines.packing_scan_photos (captured_at DESC);

-- Agar eski t3_scan_photos bo'lsa — ma'lumotni ko'chirish (bir marta)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'lines' AND table_name = 't3_scan_photos'
    ) AND NOT EXISTS (
        SELECT 1 FROM lines.packing_scan_photos LIMIT 1
    ) THEN
        INSERT INTO lines.packing_scan_photos (
            id, product_id, line_id, serial, file_path, captured_at, user_id
        )
        SELECT id, product_id, line_id, serial, file_path, captured_at, user_id
        FROM lines.t3_scan_photos
        ON CONFLICT DO NOTHING;

        PERFORM setval(
            pg_get_serial_sequence('lines.packing_scan_photos', 'id'),
            GREATEST((SELECT COALESCE(MAX(id), 1) FROM lines.packing_scan_photos), 1)
        );
    END IF;
END $$;

COMMIT;
