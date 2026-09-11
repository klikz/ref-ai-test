-- Qadoqlash liniyasi API route + ruxsatlar
-- Apply:
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/109_qadoqlash_routes.sql

BEGIN;

WITH route_comments(route, comment) AS (
    VALUES
        ('/api/lines/qadoqlash/complete', 'Qadoqlash liniyasi: lab tekshiruv, skan va chop etish'),
        ('/api/lines/qadoqlash/reprint', 'Qadoqlash liniyasi: tanlangan printerga qayta chop')
)
UPDATE auth.routes r
SET comment = route_comments.comment,
    is_hidden = false
FROM route_comments
WHERE r.route = route_comments.route;

WITH route_comments(route, comment) AS (
    VALUES
        ('/api/lines/qadoqlash/complete', 'Qadoqlash liniyasi: lab tekshiruv, skan va chop etish'),
        ('/api/lines/qadoqlash/reprint', 'Qadoqlash liniyasi: tanlangan printerga qayta chop')
)
INSERT INTO auth.routes (route, comment, is_hidden)
SELECT route_comments.route, route_comments.comment, false
FROM route_comments
WHERE NOT EXISTS (
    SELECT 1 FROM auth.routes r WHERE r.route = route_comments.route
);

-- Yi'g'ish print ruxsati bor foydalanuvchilarga qadoqlash route ruxsatini berish
WITH yigish_users AS (
    SELECT DISTINCT p.user_id
    FROM auth.permissions p
    INNER JOIN auth.routes r ON r.id = p.route_id
    WHERE r.route = '/api/lines/yigish/v2/print'
),
qadoqlash_routes AS (
    SELECT id
    FROM auth.routes
    WHERE is_hidden = false
      AND route IN (
        '/api/lines/qadoqlash/complete',
        '/api/lines/qadoqlash/reprint'
      )
)
INSERT INTO auth.permissions (user_id, route_id)
SELECT yigish_users.user_id, qadoqlash_routes.id
FROM yigish_users
CROSS JOIN qadoqlash_routes
WHERE NOT EXISTS (
    SELECT 1
    FROM auth.permissions existing
    WHERE existing.user_id = yigish_users.user_id
      AND existing.route_id = qadoqlash_routes.id
);

COMMIT;
