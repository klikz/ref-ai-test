-- Allow promoting status while copying test → prod artifacts

BEGIN;

ALTER TABLE agent.tasks DROP CONSTRAINT IF EXISTS tasks_status_check;
ALTER TABLE agent.tasks ADD CONSTRAINT tasks_status_check CHECK (status IN (
    'queued',
    'running',
    'ready_for_test',
    'testing',
    'ready_for_prod',
    'promoting',
    'promoted',
    'failed',
    'rejected'
));

COMMIT;
