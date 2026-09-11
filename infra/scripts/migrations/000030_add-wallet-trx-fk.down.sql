alter table wallet_ledger_entries
drop constraint "wallet_ledger_entries_transaction_wallet_transactions_id_fk";

alter table wallet_ledger_entries
alter column "transaction"
drop not null;