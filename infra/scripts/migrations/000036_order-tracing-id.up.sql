alter table orders add column tracing_id text not null default now()::text;
alter table orders alter column tracing_id drop default;
create index "orders_tracing_id_idx" on "orders"("tracing_id");