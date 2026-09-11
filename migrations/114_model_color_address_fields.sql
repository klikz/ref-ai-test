-- Model parameters: door/body color variants, numeric color code, Russian address
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/114_model_color_address_fields.sql

BEGIN;

ALTER TABLE production.models
    ADD COLUMN IF NOT EXISTS eshik_rangi TEXT NOT NULL DEFAULT '';

ALTER TABLE production.models
    ADD COLUMN IF NOT EXISTS rangi_eng TEXT NOT NULL DEFAULT '';

ALTER TABLE production.models
    ADD COLUMN IF NOT EXISTS korpus_rangi_shortname TEXT NOT NULL DEFAULT '';

ALTER TABLE production.models
    ADD COLUMN IF NOT EXISTS eshik_rangi_shortname TEXT NOT NULL DEFAULT '';

ALTER TABLE production.models
    ADD COLUMN IF NOT EXISTS rangi_kodi TEXT NOT NULL DEFAULT '';

ALTER TABLE production.models
    ADD COLUMN IF NOT EXISTS manzil_ru TEXT NOT NULL DEFAULT '';

COMMIT;
