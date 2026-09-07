create type wallet_type as enum('real', 'virtual');

alter table wallets
add column "type" wallet_type not null default 'real';

create index wallets_type_idx on wallets ("type");

create unique index wallets_owner_type_uidx on wallets ("owner", "type")
where
    (
        "owner" is not null
        and "type" is not null
    );