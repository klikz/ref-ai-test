-- Liniya operatsiyalari lines.lines_list.line_id ustuniga bog'lanadi (id emas).
-- Apply:
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/104_lines_list_line_id_fkeys.sql

BEGIN;

UPDATE lines.lines_list
SET line_id = id
WHERE line_id IS NULL;

ALTER TABLE lines.lines_list
    ALTER COLUMN line_id SET NOT NULL;

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT conname, conrelid::regclass AS tbl
        FROM pg_constraint
        WHERE confrelid = 'lines.lines_list'::regclass
          AND contype = 'f'
    LOOP
        EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I', r.tbl, r.conname);
    END LOOP;
END $$;

ALTER TABLE lines.auxiliary_products
    ADD CONSTRAINT auxiliary_products_line_id_fkey
    FOREIGN KEY (line_id) REFERENCES lines.lines_list(line_id);

ALTER TABLE lines.balance
    ADD CONSTRAINT balance_line_id_fkey
    FOREIGN KEY (line_id) REFERENCES lines.lines_list(line_id);

ALTER TABLE lines.balance_transactions
    ADD CONSTRAINT balance_transactions_line_id_fkey
    FOREIGN KEY (line_id) REFERENCES lines.lines_list(line_id);

ALTER TABLE lines.label_templates
    ADD CONSTRAINT label_templates_line_id_fkey
    FOREIGN KEY (line_id) REFERENCES lines.lines_list(line_id);

ALTER TABLE lines.line_responsibles
    ADD CONSTRAINT line_responsibles_line_id_fkey
    FOREIGN KEY (line_id) REFERENCES lines.lines_list(line_id);

ALTER TABLE lines.printers_v2
    ADD CONSTRAINT printers_v2_line_id_fkey
    FOREIGN KEY (line_id) REFERENCES lines.lines_list(line_id);

ALTER TABLE lines.products
    ADD CONSTRAINT products_line_id_fkey
    FOREIGN KEY (line_id) REFERENCES lines.lines_list(line_id);

ALTER TABLE production.consumption_norm_items
    ADD CONSTRAINT consumption_norm_items_consume_line_id_fkey
    FOREIGN KEY (consume_line_id) REFERENCES lines.lines_list(line_id);

ALTER TABLE production.consumption_norm_items
    ADD CONSTRAINT consumption_norm_items_receive_line_id_fkey
    FOREIGN KEY (receive_line_id) REFERENCES lines.lines_list(line_id);

ALTER TABLE production.daily_plan_days
    ADD CONSTRAINT daily_plan_days_line_id_fkey
    FOREIGN KEY (line_id) REFERENCES lines.lines_list(line_id);

ALTER TABLE production.daily_plan_items
    ADD CONSTRAINT daily_plan_items_line_id_fkey
    FOREIGN KEY (line_id) REFERENCES lines.lines_list(line_id);

ALTER TABLE ware.outcome
    ADD CONSTRAINT outcome_line_id_fkey
    FOREIGN KEY (line_id) REFERENCES lines.lines_list(line_id);

ALTER TABLE writeoff.document_items
    ADD CONSTRAINT document_items_line_id_fkey
    FOREIGN KEY (line_id) REFERENCES lines.lines_list(line_id);

ALTER TABLE writeoff.records
    ADD CONSTRAINT records_line_id_fkey
    FOREIGN KEY (line_id) REFERENCES lines.lines_list(line_id);

COMMIT;
