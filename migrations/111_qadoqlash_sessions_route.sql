-- Qadoqlash sessions/last route
-- Apply:
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/111_qadoqlash_sessions_route.sql

BEGIN;

WITH route_comments(route, comment) AS (
    VALUES
        ('/api/lines/qadoqlash/sessions/last', 'Qadoqlash liniyasi: oxirgi sessiyalar va oxirgi skan')
)
UPDATE auth.routes r
SET comment = route_comments.comment,
    is_hidden = false
FROM route_comments
WHERE r.route = route_comments.route;

WITH route_comments(route, comment) AS (
    VALUES
        ('/api/lines/qadoqlash/sessions/last', 'Qadoqlash liniyasi: oxirgi sessiyalar va oxirgi skan')
)
INSERT INTO auth.routes (route, comment, is_hidden)
SELECT route_comments.route, route_comments.comment, false
FROM route_comments
WHERE NOT EXISTS (
    SELECT 1 FROM auth.routes r WHERE r.route = route_comments.route
);

WITH qadoqlash_users AS (
    SELECT DISTINCT p.user_id
    FROM auth.permissions p
    INNER JOIN auth.routes r ON r.id = p.route_id
    WHERE r.route IN (
        '/api/lines/qadoqlash/complete',
        '/api/lines/yigish/v2/print'
    )
),
sessions_routes AS (
    SELECT id
    FROM auth.routes
    WHERE is_hidden = false
      AND route = '/api/lines/qadoqlash/sessions/last'
)
INSERT INTO auth.permissions (user_id, route_id)
SELECT qadoqlash_users.user_id, sessions_routes.id
FROM qadoqlash_users
CROSS JOIN sessions_routes
WHERE NOT EXISTS (
    SELECT 1
    FROM auth.permissions existing
    WHERE existing.user_id = qadoqlash_users.user_id
      AND existing.route_id = sessions_routes.id
);

COMMIT;
