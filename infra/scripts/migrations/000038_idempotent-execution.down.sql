alter table executions rename column order_status to status;
alter table executions rename column order_side to side;
alter table executions rename column order_type to "type";
drop index "executions_idempotency_token_idx";
alter table executions drop column idempotency_token;