alter table executions
drop constraint "executions_order_orders_id_fk";

alter table executions
drop constraint "executions_user_user_id_fk";

drop index "executions_fee_currency_idx";

drop index "executions_symbol_idx";

drop index "executions_user_idx";

drop index "executions_order_idx";

drop table executions;

drop type execution_status;