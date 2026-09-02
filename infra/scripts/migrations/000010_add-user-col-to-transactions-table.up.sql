ALTER TABLE "wallet_transactions"
ADD COLUMN "done_by" varchar(26);

CREATE INDEX "wallet_transactions_done_by_idx" ON "wallet_transactions" ("done_by")
WHERE
    "done_by" IS NOT NULL;

ALTER TABLE "wallet_transactions"
ADD CONSTRAINT "wallet_transactions_done_by_user_id_fk" FOREIGN KEY ("done_by") REFERENCES "user" ("id") ON DELETE SET NULL;