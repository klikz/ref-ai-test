-- Eshik models XLSX export/import routes
-- Apply:
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/124_eshik_xlsx_routes.sql

BEGIN;

WITH route_comments(route, comment) AS (
    VALUES
        ('/api/production/eshik/export', 'Eshik modellari XLSX export'),
        ('/api/production/eshik/template', 'Eshik modellari XLSX shablon'),
        ('/api/production/eshik/import', 'Eshik modellari XLSX import')
)
UPDATE auth.routes r
SET comment = route_comments.comment,
    is_hidden = false
FROM route_comments
WHERE r.route = route_comments.route;

WITH route_comments(route, comment) AS (
    VALUES
        ('/api/production/eshik/export', 'Eshik modellari XLSX export'),
        ('/api/production/eshik/template', 'Eshik modellari XLSX shablon'),
        ('/api/production/eshik/import', 'Eshik modellari XLSX import')
)
INSERT INTO auth.routes (route, comment, is_hidden)
SELECT route_comments.route, route_comments.comment, false
FROM route_comments
WHERE NOT EXISTS (
    SELECT 1 FROM auth.routes r WHERE r.route = route_comments.route
);

-- Eshik catalog CRUD ruxsati bor foydalanuvchilarga export/import berish
WITH eshik_users AS (
    SELECT DISTINCT p.user_id
    FROM auth.permissions p
    INNER JOIN auth.routes r ON r.id = p.route_id
    WHERE r.route = '/api/production/eshik/all'
),
eshik_xlsx_routes AS (
    SELECT id
    FROM auth.routes
    WHERE is_hidden = false
      AND route IN (
        '/api/production/eshik/export',
        '/api/production/eshik/template',
        '/api/production/eshik/import'
      )
)
INSERT INTO auth.permissions (user_id, route_id)
SELECT eshik_users.user_id, eshik_xlsx_routes.id
FROM eshik_users
CROSS JOIN eshik_xlsx_routes
WHERE NOT EXISTS (
    SELECT 1
    FROM auth.permissions existing
    WHERE existing.user_id = eshik_users.user_id
      AND existing.route_id = eshik_xlsx_routes.id
);

COMMIT;
