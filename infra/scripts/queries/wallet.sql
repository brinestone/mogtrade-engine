-- name: FindWalletTransactionById :one
select
    *
from
    wallet_transactions
where
    id = $1;

-- name: CreateTransactionLedgerEntry :exec
insert into
    wallet_ledger_entries (wallet, "transaction", notes)
values
    ($1, $2, $3);

-- name: UpdateTransactionStatusById :exec
update wallet_transactions
set
    status = $1,
    updated_at = now()
where
    id = $2;

-- name: IdempotencyKeyExists :one
select
    exists (
        select
            1
        from
            wallet_transactions
        where
            idempotency_token = $1
    );

-- name: RecordWalletTransaction :exec
insert into
    wallet_transactions (
        id,
        "value",
        src,
        dest,
        intent,
        extra_data,
        idempotency_token,
        done_by,
        tracing_id,
        currency,
        exchange_rate
    )
values
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: RefreshWalletSnapshots :exec
refresh materialized view wallet_snapshots;

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