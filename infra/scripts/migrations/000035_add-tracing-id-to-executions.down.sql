-- Revert tracing_id column addition
DROP INDEX IF EXISTS "executions_tracing_id_idx";
ALTER TABLE executions DROP COLUMN tracing_id;