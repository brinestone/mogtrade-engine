ALTER TABLE "wallet_transactions"
DROP CONSTRAINT "wallet_transactions_don_by_user_id_fk";

DROP INDEX "wallet_transactions_done_by_idx";

ALTER TABLE "wallet_transactions"
DROP COLUMN "done_by" CASCADE;