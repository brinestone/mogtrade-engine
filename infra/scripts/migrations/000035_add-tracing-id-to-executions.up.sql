-- Add tracing_id column to executions table for end-to-end traceability
ALTER TABLE executions ADD COLUMN tracing_id varchar(26) NOT NULL;
CREATE UNIQUE INDEX "executions_tracing_id_idx" ON executions (tracing_id);