alter table orders
drop column fee;

alter table orders
add column bid_transaction varchar(26);

alter table orders
add column ask_wallet varchar(26);

create index "orders_bid_transaction_idx" on "orders" ("bid_transaction");

create index "orders_ask_wallet_idx" on "orders" ("ask_wallet");

alter table orders
add constraint "orders_bid_transaction_wallet_transactions_id_fk" foreign key ("bid_transaction") references "wallet_transactions" ("id") on delete set null;

alter table orders
add constraint "orders_ask_wallet_wallets_id_fk" foreign key ("ask_wallet") references "wallets" ("id") on delete set null;