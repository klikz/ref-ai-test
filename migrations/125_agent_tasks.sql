-- Agent tasks (owner-only UI/API)
-- Apply:
--   psql "... dbname=ref-test ..." -f migrations/125_agent_tasks.sql

BEGIN;

CREATE SCHEMA IF NOT EXISTS agent;

CREATE TABLE IF NOT EXISTS agent.tasks (
    id              BIGSERIAL PRIMARY KEY,
    prompt          TEXT NOT NULL,
    origin          TEXT NOT NULL DEFAULT 'server'
                    CHECK (origin IN ('pc', 'server')),
    status          TEXT NOT NULL DEFAULT 'queued'
                    CHECK (status IN (
                        'queued',
                        'running',
                        'ready_for_test',
                        'testing',
                        'ready_for_prod',
                        'promoted',
                        'failed',
                        'rejected'
                    )),
    created_by      INTEGER NOT NULL,
    git_sha         TEXT NOT NULL DEFAULT '',
    log_text        TEXT NOT NULL DEFAULT '',
    error_text      TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_tasks_status ON agent.tasks (status);
CREATE INDEX IF NOT EXISTS idx_agent_tasks_created_at ON agent.tasks (created_at DESC);

WITH route_comments(route, comment) AS (
    VALUES
        ('/api/agent/access', 'Agent moduliga kirish (owner)'),
        ('/api/agent/tasks', 'Agent vazifalar ro''yxati'),
        ('/api/agent/tasks/create', 'Agent vazifa yaratish'),
        ('/api/agent/tasks/get', 'Agent vazifa tafsiloti'),
        ('/api/agent/tasks/approve-test', 'Agent vazifani testga chiqarish'),
        ('/api/agent/tasks/approve-prod', 'Agent vazifani prodga chiqarish'),
        ('/api/agent/tasks/reject', 'Agent vazifani rad etish')
)
INSERT INTO auth.routes (route, comment, is_hidden)
SELECT route_comments.route, route_comments.comment, false
FROM route_comments
WHERE NOT EXISTS (
    SELECT 1 FROM auth.routes r WHERE r.route = route_comments.route
);

-- access: barcha faol userlar (faqat allowed true/false qaytaradi)
INSERT INTO auth.permissions (user_id, route_id)
SELECT u.id, r.id
FROM auth.users u
CROSS JOIN auth.routes r
WHERE u.status = true
  AND r.route = '/api/agent/access'
  AND NOT EXISTS (
      SELECT 1 FROM auth.permissions p
      WHERE p.user_id = u.id AND p.route_id = r.id
  );

-- qolgan agent API: faqat admin (runtime da AGENT_OWNER_USER_IDS qo'shimcha filtr)
WITH agent_routes AS (
    SELECT id FROM auth.routes
    WHERE route LIKE '/api/agent/%'
      AND route <> '/api/agent/access'
),
admin_users AS (
    SELECT u.id AS user_id
    FROM auth.users u
    INNER JOIN auth.roles r ON r.id = u.role_id
    WHERE u.status = true
      AND lower(r.name) = 'admin'
)
INSERT INTO auth.permissions (user_id, route_id)
SELECT au.user_id, ar.id
FROM admin_users au
CROSS JOIN agent_routes ar
WHERE NOT EXISTS (
    SELECT 1
    FROM auth.permissions p
    WHERE p.user_id = au.user_id
      AND p.route_id = ar.id
);

COMMIT;
