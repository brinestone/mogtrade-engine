drop index executions_idempotency_token_order_uidx;

CREATE INDEX "executions_idempotency_token_idx" ON "executions" ("idempotency_token");