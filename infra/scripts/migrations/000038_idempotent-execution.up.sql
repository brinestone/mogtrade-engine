alter table executions rename column status to order_status;
alter table executions rename column side to order_side;
alter table executions rename column "type" to order_type;
alter table executions add column idempotency_token text not null default now()::text;
alter table executions alter column idempotency_token drop default;
create index "executions_idempotency_token_idx" on "executions"("idempotency_token");