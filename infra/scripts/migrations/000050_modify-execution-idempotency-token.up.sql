drop index executions_idempotency_token_idx;
create unique index executions_idempotency_token_order_uidx on executions(idempotency_token, "order");
