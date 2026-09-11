drop index if exists "wallet_transactions_type_idx";

alter table "wallet_transactions"
drop column "type";

drop type "wallet_transaction_type";