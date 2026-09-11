alter table wallet_transactions
drop constraint wallet_transactions_idempotency_token_key;

drop index if exists "wallet_transactions_idempotency_token_key";

create index "wallet_transactions_idempotency_token_idx" on "wallet_transactions" ("idempotency_token");