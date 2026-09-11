-- Split models.door_code into freeze_door_code + ref_door_code;
-- qadoqlash stores both scanned door serials.
-- Apply:
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/added/117_model_door_codes.sql

BEGIN;

ALTER TABLE production.models
    ADD COLUMN IF NOT EXISTS freeze_door_code TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ref_door_code TEXT NOT NULL DEFAULT '';

UPDATE production.models
SET freeze_door_code = COALESCE(NULLIF(TRIM(freeze_door_code), ''), COALESCE(door_code, ''), ''),
    ref_door_code = COALESCE(NULLIF(TRIM(ref_door_code), ''), COALESCE(door_code, ''), '')
WHERE COALESCE(NULLIF(TRIM(freeze_door_code), ''), NULLIF(TRIM(ref_door_code), ''), '') = '';

ALTER TABLE lines.product_params
    ADD COLUMN IF NOT EXISTS freeze_door_serial TEXT,
    ADD COLUMN IF NOT EXISTS ref_door_serial TEXT;

UPDATE lines.product_params
SET freeze_door_serial = COALESCE(NULLIF(TRIM(freeze_door_serial), ''), NULLIF(TRIM(door_serial), '')),
    ref_door_serial = COALESCE(NULLIF(TRIM(ref_door_serial), ''), NULLIF(TRIM(door_serial), ''))
WHERE COALESCE(door_serial, '') <> ''
  AND (
    COALESCE(NULLIF(TRIM(freeze_door_serial), ''), '') = ''
    OR COALESCE(NULLIF(TRIM(ref_door_serial), ''), '') = ''
  );

CREATE UNIQUE INDEX IF NOT EXISTS product_params_freeze_door_serial_un
    ON lines.product_params (freeze_door_serial)
    WHERE freeze_door_serial IS NOT NULL AND freeze_door_serial <> '';

CREATE UNIQUE INDEX IF NOT EXISTS product_params_ref_door_serial_un
    ON lines.product_params (ref_door_serial)
    WHERE ref_door_serial IS NOT NULL AND ref_door_serial <> '';

COMMIT;
