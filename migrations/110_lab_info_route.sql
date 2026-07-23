-- Laboratoriya API route
-- Apply:
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/110_lab_info_route.sql

BEGIN;

WITH route_comments(route, comment) AS (
    VALUES
        ('/api/lab/info', 'Laboratoriya (VTM) BxData — serial bo''yicha test natijalari'),
        ('/api/serial/info', 'Serial/kompressor bo''yicha mahsulot, params va skan surati')
)
UPDATE auth.routes r
SET comment = route_comments.comment,
    is_hidden = false
FROM route_comments
WHERE r.route = route_comments.route;

WITH route_comments(route, comment) AS (
    VALUES
        ('/api/lab/info', 'Laboratoriya (VTM) BxData — serial bo''yicha test natijalari'),
        ('/api/serial/info', 'Serial/kompressor bo''yicha mahsulot, params va skan surati')
)
INSERT INTO auth.routes (route, comment, is_hidden)
SELECT route_comments.route, route_comments.comment, false
FROM route_comments
WHERE NOT EXISTS (
    SELECT 1 FROM auth.routes r WHERE r.route = route_comments.route
);

COMMIT;
