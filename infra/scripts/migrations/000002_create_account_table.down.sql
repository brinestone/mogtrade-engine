TRUNCATE TABLE "account";
ALTER TABLE "account" DROP CONSTRAINT "account_user_id_user_id_fk";
DROP INDEX "account_user_id_idx";
DROP INDEX "account_issuer_account_id_uidx";
DROP TYPE IF EXISTS account_provider;

DROP TABLE "account";