drop index wallets_owner_type_uidx;

drop index wallets_type_idx;

alter table wallets
drop column "type";

drop type "wallet_type";