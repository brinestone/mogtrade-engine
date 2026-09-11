drop index wallet_snapshots_wallet_id_idx;

drop materialized view wallet_snapshots;

CREATE MATERIALIZED VIEW
    public.wallet_snapshots TABLESPACE pg_default AS
WITH
    transaction_deltas AS (
        SELECT
            wallet_transactions.src AS wallet_id,
            '-1'::integer::numeric * wallet_transactions.value AS net_change,
            wallet_transactions.recorded_at
        FROM
            wallet_transactions
        WHERE
            wallet_transactions.src IS NOT NULL
        UNION ALL
        SELECT
            wallet_transactions.dest AS wallet_id,
            wallet_transactions.value AS net_change,
            wallet_transactions.recorded_at
        FROM
            wallet_transactions
        WHERE
            wallet_transactions.dest IS NOT NULL
    ),
    wallet_totals AS (
        SELECT
            transaction_deltas.wallet_id,
            COALESCE(sum(transaction_deltas.net_change), 0::numeric) AS current_balance,
            count(*) AS total_transactions,
            max(transaction_deltas.recorded_at) AS last_activity_at
        FROM
            transaction_deltas
        GROUP BY
            transaction_deltas.wallet_id
    )
SELECT
    w.id AS wallet_id,
    w.owner AS owner_id,
    w.starting_balance,
    (
        w.starting_balance + COALESCE(t.current_balance, 0::numeric)
    )::numeric(18, 4) AS current_balance,
    COALESCE(t.total_transactions, 0::bigint) AS total_transactions,
    t.last_activity_at,
    now() AS snapshot_created_at,
    w.type AS wallet_type
FROM
    wallets w
    LEFT JOIN wallet_totals t ON w.id::text = t.wallet_id::text
WITH
    DATA;

-- View indexes:
CREATE UNIQUE INDEX wallet_snapshots_wallet_id_idx ON public.wallet_snapshots USING btree (wallet_id);