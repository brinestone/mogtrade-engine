ALTER TABLE "wallet_transactions"
ADD COLUMN "tracing_id" TEXT NOT NULL DEFAULT now()::TEXT;

CREATE INDEX "wallet_transactions_tracing_id_idx" ON "wallet_transactions" ("tracing_id");