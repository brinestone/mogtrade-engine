DROP INDEX "account_issuer_account_id_uidx";

alter table account
drop column issuer;

create unique index "account_provider_user_id_uidx" ON "account" ("provider", "account_id");