drop index wallet_snapshots_wallet_id_idx;
drop materialized view if exists wallet_snapshots;
alter table wallet_transactions drop column exchange_rate;
alter table wallet_transactions drop column currency;

CREATE MATERIALIZED VIEW
    wallet_snapshots AS
WITH
    transaction_deltas AS (
        -- Deduct from source wallet
        SELECT
            src AS wallet_id,
            - value AS net_change,
            recorded_at
        FROM
            wallet_transactions
        WHERE
            src IS NOT NULL
        UNION ALL
        -- Add to destination wallet
        SELECT
            dest AS wallet_id,
            value AS net_change,
            recorded_at
        FROM
            wallet_transactions
        WHERE
            dest IS NOT NULL
    ),
    wallet_totals AS (
        SELECT
            wallet_id,
            COALESCE(SUM(net_change), 0) AS current_balance,
            COUNT(*) AS total_transactions,
            MAX(recorded_at)::TIMESTAMPTZ AS last_activity_at
        FROM
            transaction_deltas
        GROUP BY
            wallet_id
    )
SELECT
    w.id AS wallet_id,
    w.owner AS owner_id,
    w.starting_balance,
    (
        w.starting_balance + COALESCE(t.current_balance, 0)
    )::numeric(18, 4) AS current_balance,
    COALESCE(t.total_transactions, 0) AS total_transactions,
    t.last_activity_at,
    NOW()::TIMESTAMPTZ AS snapshot_created_at
FROM
    wallets w
    LEFT JOIN wallet_totals t ON w.id = t.wallet_id;

CREATE UNIQUE INDEX wallet_snapshots_wallet_id_idx ON wallet_snapshots (wallet_id);