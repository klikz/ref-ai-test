-- Line display order + new "Eshik yig'uv va eshikka PPU quyish" uchastka
-- Apply:
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/added/122_lines_sort_order.sql

BEGIN;

ALTER TABLE lines.lines_list
    ADD COLUMN IF NOT EXISTS sort_order INT NOT NULL DEFAULT 0;

INSERT INTO lines.lines_list (id, line_id, name, status, folder_name, sort_order)
OVERRIDING SYSTEM VALUE
SELECT 20, 20, 'Eshik yig''uv va eshikka PPU quyish uchastkasi', true, 'eshik_ppu', 4
WHERE NOT EXISTS (
    SELECT 1 FROM lines.lines_list WHERE line_id = 20
);

UPDATE lines.lines_list SET sort_order = 1  WHERE line_id = 13;
UPDATE lines.lines_list SET sort_order = 2  WHERE line_id = 11;
UPDATE lines.lines_list SET sort_order = 3  WHERE line_id = 14;
UPDATE lines.lines_list SET sort_order = 4  WHERE line_id = 20;
UPDATE lines.lines_list SET sort_order = 5  WHERE line_id = 15;
UPDATE lines.lines_list SET sort_order = 6  WHERE line_id = 16;
UPDATE lines.lines_list SET sort_order = 7  WHERE line_id = 1;
UPDATE lines.lines_list SET sort_order = 8  WHERE line_id = 17;
UPDATE lines.lines_list SET sort_order = 9  WHERE line_id = 18;
UPDATE lines.lines_list SET sort_order = 10 WHERE line_id = 12;
UPDATE lines.lines_list SET sort_order = 11 WHERE line_id = 19;

-- Other lines (T1/T2/… etc.) after the workshop list
UPDATE lines.lines_list
SET sort_order = 100 + line_id
WHERE sort_order = 0;

COMMIT;
