drop index if exists "wallet_transactions_idempotency_token_uidx";

create index "wallet_transactions_idempotency_token_idx" on "wallet_transactions" ("idempotency_token");