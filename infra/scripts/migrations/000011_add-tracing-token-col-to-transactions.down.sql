DROP INDEX "wallet_transactions_tracing_id_idx";

ALTER TABLE "wallet_transactions"
DROP COLUMN "tracing_id";