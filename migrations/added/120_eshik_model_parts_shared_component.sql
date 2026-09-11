-- Allow one component to be used across multiple eshik models
-- Apply:
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/120_eshik_model_parts_shared_component.sql

BEGIN;

ALTER TABLE production.eshik_model_parts
    DROP CONSTRAINT IF EXISTS eshik_model_parts_component_un;

COMMIT;
