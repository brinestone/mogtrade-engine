LTER TABLE "wallet_event_ledger"
DROP CONSTRAINT "wallet_event_ledger_wallet_wallets_id_fk";

DROP INDEX "wallet_event_ledger_transaction_idx";

DROP INDEX "wallet_event_ledger_wallet_idx";

DROP TABLE wallet_event_ledger;

ALTER TABLE "wallet_transactions"
DROP CONSTRAINT wallet_transactions_dest_wallets_id_fk;

ALTER TABLE wallet_transactions
DROP CONSTRAINT wallet_transactions_src_wallets_id_fk;

DROP INDEX wallet_transactions_src_dest_idx;

DROP INDEX wallet_transactions_src_idx;

DROP INDEX wallet_transactions_type_idx;

DROP TABLE wallet_transactions;

DROP TYPE IF EXISTS wallet_transaction_status;

DROP TYPE IF EXISTS wallet_transaction_type;

ALTER TABLE wallets
DROP CONSTRAINT wallets_owner_user_id_fk;

DROP INDEX wallets_owner_uidx;

DROP TABLE wallets;