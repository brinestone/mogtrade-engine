alter table wallet_ledger_entries
alter column "transaction"
set not null;

alter table wallet_ledger_entries
add constraint "wallet_ledger_entries_transaction_wallet_transactions_id_fk" foreign key ("transaction") references "wallet_transactions" ("id") on delete cascade;