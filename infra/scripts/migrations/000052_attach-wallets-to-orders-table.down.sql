alter table orders
drop constraint "orders_ask_wallet_wallets_id_fk";

alter table orders
drop constraint "orders_bid_transaction_wallet_transactions_id_fk";

drop index "orders_ask_wallet_idx";

drop index "orders_bid_transaction_idx";

alter table orders
drop column ask_wallet;

alter table orders
drop column bid_transaction;

alter table orders
add column fee numeric(18, 4) not null default 0;

alter table orders
alter column fee
drop default;