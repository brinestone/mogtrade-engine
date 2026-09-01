CREATE TABLE
    wallets (
        id varchar(26) PRIMARY KEY,
        "owner" varchar(26),
        starting_balance numeric(18, 4) DEFAULT 0.0
    );

CREATE UNIQUE INDEX "wallets_owner_uidx" ON "wallets" ("owner")
WHERE
    "owner" IS NOT NULL;

ALTER TABLE wallets
ADD CONSTRAINT "wallets_owner_user_id_fk" FOREIGN KEY ("owner") REFERENCES "user" ("id") ON DELETE SET NULL;

CREATE TYPE "wallet_transaction_type" AS ENUM('credit', 'debit');

CREATE TYPE "wallet_transaction_status" AS ENUM('processing', 'processed', 'cancelled', 'failed');

CREATE TABLE
    wallet_transactions (
        id varchar(26) PRIMARY KEY,
        "type" wallet_transaction_type NOT NULL,
        "value" numeric(18, 4) NOT NULL,
        src varchar(26),
        dest varchar(26),
        motive text,
        recorded_at TIMESTAMPTZ not null default now(),
        updated_at TIMESTAMPTZ not null default now(),
        extra_data JSONB,
        idempotency_token text not null unique
    );

CREATE INDEX "wallet_transactions_type_idx" ON "wallet_transactions" ("type");

CREATE INDEX "wallet_transactions_src_idx" ON "wallet_transactions" ("src")
WHERE
    src is not null;

CREATE INDEX "wallet_transactions_dest_idx" ON "wallet_transactions" ("dest")
WHERE
    dest is not null;

CREATE INDEX "wallet_transactions_src_dest_idx" ON "wallet_transactions" ("src", "dest");

ALTER TABLE "wallet_transactions"
ADD CONSTRAINT "wallet_transactions_src_wallets_id_fk" FOREIGN KEY ("src") REFERENCES "wallets" ("id") ON DELETE SET NULL;

ALTER TABLE "wallet_transactions"
ADD CONSTRAINT "wallet_transactions_dest_wallets_id_fk" FOREIGN KEY ("dest") REFERENCES "wallets" ("id") ON DELETE SET NULL;

CREATE TABLE
    wallet_event_ledger (
        wallet varchar(26),
        notes text not null,
        recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
        "transaction" varchar(26)
    );

CREATE INDEX "wallet_event_ledger_wallet_idx" ON "wallet_event_ledger" ("wallet")
WHERE
    "wallet" IS NOT NULL;

CREATE INDEX "wallet_event_ledger_transaction_idx" ON "wallet_event_ledger" ("transaction")
WHERE
    "transaction" IS NOT NULL;

ALTER TABLE "wallet_event_ledger"
ADD CONSTRAINT "wallet_event_ledger_wallet_wallets_id_fk" FOREIGN KEY ("wallet") REFERENCES "wallets" ("id") ON DELETE SET NULL;