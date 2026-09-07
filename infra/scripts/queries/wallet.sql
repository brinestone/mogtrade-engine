-- name: CreateVirtualWallet :exec
insert into
    wallets ("type", id, "owner", starting_balance)
values
    ('virtual', $1, $2, $3);

-- name: UserHasWallet :one
select
    exists (
        select
            1
        from
            wallets
        where
            "owner" = $1
            and "type" = $2
    );

-- name: CreateRealWallet :exec
insert into
    wallets (id, "owner", starting_balance)
values
    ($1, $2, $3);

-- name: FindWalletSnapshotByOwnerId :one
SELECT
    *
FROM
    wallet_snapshots
WHERE
    owner_id = $1
LIMIT
    1;