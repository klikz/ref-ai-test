-- Model parameter: Nominal tok kuchi (A)
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/added/119_model_nominal_tok_kuchi_a.sql

BEGIN;

ALTER TABLE production.models
    ADD COLUMN IF NOT EXISTS nominal_tok_kuchi_a TEXT NOT NULL DEFAULT '';

COMMIT;
