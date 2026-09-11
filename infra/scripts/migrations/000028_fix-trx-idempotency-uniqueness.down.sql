drop index if exists "wallet_transactions_idempotency_token_idx";

alter table wallet_transactions
add unique (idempotency_token);