-- Add freeze/ref door serials on lines.products (chiqgan mahsulotlar).
-- Packing (qadoqlash) scan success updates these by product serial.
-- Apply:
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/added/118_products_door_serials.sql

BEGIN;

ALTER TABLE lines.products
    ADD COLUMN IF NOT EXISTS freeze_door_serial TEXT,
    ADD COLUMN IF NOT EXISTS ref_door_serial TEXT;

-- Ensure product_params columns exist (idempotent with 117)
ALTER TABLE lines.product_params
    ADD COLUMN IF NOT EXISTS freeze_door_serial TEXT,
    ADD COLUMN IF NOT EXISTS ref_door_serial TEXT;

CREATE INDEX IF NOT EXISTS idx_products_freeze_door_serial
    ON lines.products (freeze_door_serial)
    WHERE freeze_door_serial IS NOT NULL AND freeze_door_serial <> '';

CREATE INDEX IF NOT EXISTS idx_products_ref_door_serial
    ON lines.products (ref_door_serial)
    WHERE ref_door_serial IS NOT NULL AND ref_door_serial <> '';

COMMIT;
