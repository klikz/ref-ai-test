-- Label template print settings required by V2 print engine (density/speed/gap/defaults)
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/113_label_template_print_settings.sql

BEGIN;

ALTER TABLE lines.label_templates
    ADD COLUMN IF NOT EXISTS density INTEGER NOT NULL DEFAULT 8;

ALTER TABLE lines.label_templates
    ADD COLUMN IF NOT EXISTS speed INTEGER NOT NULL DEFAULT 4;

ALTER TABLE lines.label_templates
    ADD COLUMN IF NOT EXISTS gap_mm DOUBLE PRECISION NOT NULL DEFAULT 2;

ALTER TABLE lines.label_templates
    ADD COLUMN IF NOT EXISTS use_printer_defaults BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE lines.label_templates
    ADD COLUMN IF NOT EXISTS size_only BOOLEAN NOT NULL DEFAULT false;

COMMIT;
