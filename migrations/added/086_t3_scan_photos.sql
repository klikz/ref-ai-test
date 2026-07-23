-- Packing / line scan photos from IP camera
-- Apply: psql -U postgres -d ac -f migrations/086_t3_scan_photos.sql
-- Yangi nom: lines.packing_scan_photos (qarang: migrations/108_packing_scan_photos.sql)

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

COMMIT;
