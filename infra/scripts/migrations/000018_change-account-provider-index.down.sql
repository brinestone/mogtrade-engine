drop index account_provider_user_id_uidx;

alter table account
add column issuer text default '' not null;

create unique index account_issuer_account_id_uidx on account (issuer, account_id);