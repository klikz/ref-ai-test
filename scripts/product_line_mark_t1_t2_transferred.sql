-- Bir martalik: FAQAT T1 va T2 → transferred.
-- T3 (line_id = 6) umuman o'zgartirilmaydi. INSERT yo'q.
-- Oldin scripts/product_line_balance_report.sql bilan tekshiring.
--
-- Qo'llash:
--   psql -U postgres -d ac -f scripts/product_line_mark_t1_t2_transferred.sql

BEGIN;

-- T1 (line_id = 4)
UPDATE lines.products
SET status = 'transferred',
    transferred_at = COALESCE(transferred_at, NOW())
WHERE line_id = 4
  AND status = 'active';

-- T2 (line_id = 5)
UPDATE lines.products
SET status = 'transferred',
    transferred_at = COALESCE(transferred_at, NOW())
WHERE line_id = 5
  AND status = 'active';

COMMIT;
