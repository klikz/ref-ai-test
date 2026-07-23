-- Brigadir: nakladnoma pozitsiyalarini qabul qilish (is_received, received_at, received_by)
-- Apply:
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/105_brigadir_delivery_receive.sql

BEGIN;

ALTER TABLE ware.delivery_note_items
    ADD COLUMN IF NOT EXISTS is_received BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE ware.delivery_note_items
    ADD COLUMN IF NOT EXISTS received_at TIMESTAMPTZ;

ALTER TABLE ware.delivery_note_items
    ADD COLUMN IF NOT EXISTS received_by INT REFERENCES auth.users(id);

CREATE INDEX IF NOT EXISTS idx_ware_delivery_note_items_received
    ON ware.delivery_note_items (is_received, line_id)
    WHERE is_ready = true;

COMMIT;
