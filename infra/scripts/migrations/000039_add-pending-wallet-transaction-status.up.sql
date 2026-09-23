drop index wallet_snapshots_wallet_id_idx;

drop materialized view wallet_snapshots;

alter table wallet_transactions
alter column status
drop default;

alter table wallet_transactions
alter column status
type text using status::text;

drop type transaction_status;

create type transaction_status as enum(
    'processing',
    'failed',
    'cancelled',
    'completed',
    'created'
);

alter table wallet_transactions
alter column status
type transaction_status using status::transaction_status;

alter table wallet_transactions
alter column status
set default 'created'::transaction_status;

create materialized view
    wallet_snapshots as
with
    transaction_deltas as (
        select
            src as wallet_id,
            (
                case
                    when status in ('completed', 'created', 'processing') then '-1'::numeric * "value"
                    else 0
                end
            )::numeric(18, 4) as net_change,
            recorded_at
        from
            wallet_transactions
        where
            src is not null
        union all
        select
            dest as walle_id,
            (
                case
                    when status in ('completed', 'created', 'processing') then "value"
                    else 0
                end
            )::numeric(18, 4) as net_change,
            recorded_at
        from
            wallet_transactions
        where
            dest is not null
    ),
    wallet_totals as (
        select
            wallet_id,
            coalesce(sum(net_change), 0)::numeric as current_balance,
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
    coalesce(starting_balance, 0)::numeric(18, 4) as starting_balance,
    (
        w.starting_balance + coalesce(t.current_balance, 0::numeric)
    )::numeric(18, 4) as current_balance,
    coalesce(t.total_transactions, 0)::bigint as total_transactions,
    t.last_activity_at,
    now()::timestamptz as snapshot_created_at,
    w."type" as wallet_type
from
    wallets w
    left join wallet_totals t on w.id = t.wallet_id;

create unique index "wallet_snapshots_wallet_id_idx" on "wallet_snapshots" ("wallet_id");