-- name: FindWalletSnapshotByOwnerId :one
SELECT
    *
FROM
    wallet_snapshots
WHERE
    owner_id = $1
LIMIT
    1;