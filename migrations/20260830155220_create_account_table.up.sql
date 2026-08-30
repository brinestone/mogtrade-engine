CREATE TABLE account(
    id: varchar(26) PRIMARY KEY,
    issuer text NOT NULL,
    account_id TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    user_id varchar(26) NOT NULL,
    access_token TEXT,
    refresh_token TEXT,
    id_token TEXT,
    access_token_expires_at TIMESTAMP,
    refresh_token_expires_at TIMESTAMP,
    scope TEXT,
    "password" TEXT,
    created_at timestamp not null default now(),
    updated_at timestamp not null default now()
);

CREATE UNIQUE INDEX "account_issuer_account_id_uidx" ON "account" ("issuer","account_id");
CREATE INDEX "account_user_id_idx" ON "account" ("user_id");
ALTER TABLE "account" ADD CONSTRAINT "account_user_id_user_id_fk" FOREIGN KEY ("user_id") REFERENCES "user"("id") ON DELETE CASCADE;