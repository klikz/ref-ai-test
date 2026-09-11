-- Move /eshik operational line from line_id=11 to line_id=20
-- 11 stays "Eshiklarni kesish va formalash uchastkasi" (catalog)
-- 20 is "Eshik yig'uv va eshikka PPU quyish uchastkasi" (print/plan/balance)
-- Apply:
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/123_eshik_line_id_20.sql

BEGIN;

INSERT INTO lines.lines_list (id, line_id, name, status, folder_name, sort_order)
OVERRIDING SYSTEM VALUE
SELECT 20, 20, 'Eshik yig''uv va eshikka PPU quyish uchastkasi', true, 'eshik', 4
WHERE NOT EXISTS (
    SELECT 1 FROM lines.lines_list WHERE line_id = 20
);

UPDATE lines.lines_list
SET name = 'Eshik yig''uv va eshikka PPU quyish uchastkasi',
    folder_name = 'eshik',
    status = true,
    sort_order = 4
WHERE line_id = 20;

UPDATE lines.lines_list
SET name = 'Eshiklarni kesish va formalash uchastkasi',
    folder_name = 'eshik_kesish',
    status = true,
    sort_order = 2
WHERE line_id = 11;

-- Operational data that belonged to the old /eshik line (11 → 20)
UPDATE lines.printers_v2 SET line_id = 20 WHERE line_id = 11;
UPDATE lines.label_templates SET line_id = 20 WHERE line_id = 11;
UPDATE lines.line_responsibles SET line_id = 20 WHERE line_id = 11;

UPDATE production.daily_plan_days SET line_id = 20 WHERE line_id = 11
  AND NOT EXISTS (
      SELECT 1 FROM production.daily_plan_days d2
      WHERE d2.plan_date = production.daily_plan_days.plan_date AND d2.line_id = 20
  );
DELETE FROM production.daily_plan_days WHERE line_id = 11;

UPDATE production.daily_plan_items SET line_id = 20 WHERE line_id = 11;

UPDATE lines.auxiliary_products SET line_id = 20 WHERE line_id = 11;
UPDATE lines.balance SET line_id = 20 WHERE line_id = 11;
UPDATE lines.balance_transactions SET line_id = 20 WHERE line_id = 11;

UPDATE lines.print_v2_events SET line_id = 20 WHERE line_id = 11;

COMMIT;
