create unique index "wallets_owner_uidx" on "wallets" ("owner")
where
    "owner" is not null;