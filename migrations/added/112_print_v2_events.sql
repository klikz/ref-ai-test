-- Print V2 events / metrics (port from ac-main print engine)
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/112_print_v2_events.sql

BEGIN;

CREATE TABLE IF NOT EXISTS lines.print_v2_events (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ok BOOLEAN NOT NULL,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    line_id INTEGER NULL,
    line_name TEXT NOT NULL DEFAULT '',
    printer_v2_id INTEGER NULL,
    printer_name TEXT NOT NULL DEFAULT '',
    template_id INTEGER NULL,
    print_language TEXT NOT NULL DEFAULT '',
    effective_language TEXT NOT NULL DEFAULT '',
    serial TEXT NOT NULL DEFAULT '',
    stage TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    error_detail TEXT NOT NULL DEFAULT '',
    meta JSONB NULL
);

CREATE INDEX IF NOT EXISTS print_v2_events_created_at_idx
    ON lines.print_v2_events (created_at DESC);

CREATE INDEX IF NOT EXISTS print_v2_events_ok_created_at_idx
    ON lines.print_v2_events (ok, created_at DESC);

CREATE INDEX IF NOT EXISTS print_v2_events_line_id_created_at_idx
    ON lines.print_v2_events (line_id, created_at DESC);

COMMIT;
