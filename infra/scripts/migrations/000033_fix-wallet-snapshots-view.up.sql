drop index if exists wallet_snapshots_wallet_id_idx;

drop materialized view wallet_snapshots;

create materialized view
    wallet_snapshots as
with
    transaction_deltas as (
        select
            src as wallet_id,
            case
                when status = 'completed' then -1 * "value"
                else 0
            end as net_change,
            recorded_at
        from
            wallet_transactions
        where
            src is not null
        union all
        select
            dest as wallet_id,
            case
                when status = 'completed' then "value"
                else 0
            end as net_change,
            recorded_at
        from
            wallet_transactions
        where
            dest is not null
    ),
    wallet_totals as (
        select
            wallet_id,
            coalesce(sum(net_change), 0) as current_balance,
            count(*) as total_transactions,
            max(recorded_at) as last_activity_at
        from
            transaction_deltas
        group by
            wallet_id
    )
select
    w.id as wallet_id,
    w."owner" as owner_id,
    w.starting_balance,
    (
        w.starting_balance + coalesce(t.current_balance, 0)
    )::numeric(18, 4) as current_balance,
    coalesce(t.total_transactions, 0) as total_transactions,
    t.last_activity_at,
    now() as snapshot_created_at,
    w."type" as wallet_type
from
    wallets w
    left join wallet_totals t on w.id = t.wallet_id;

create unique index wallet_snapshots_wallet_id_idx on wallet_snapshots (wallet_id);