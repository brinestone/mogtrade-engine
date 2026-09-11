CREATE TYPE "wallet_transaction_type" AS ENUM('credit', 'debit');

alter table wallet_transactions
add column "type" "wallet_transaction_type" not null default 'credit';

alter table wallet_transactions
alter column "type"
drop default;

create index "wallet_transactions_type_idx" on "wallet_transactions" ("type");