drop index if exists "wallet_transactions_idempotency_token_idx";

create unique index "wallet_transactions_idempotency_token_uidx" on "wallet_transactions" ("idempotency_token");
