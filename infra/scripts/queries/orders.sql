-- name: GetOrderWalletType :one
select
    bw."type" as wallet_type
from
    orders o
    inner join wallet_transactions wt on o.bid_transaction = wt.id
    inner join wallets bw on wt.src = bw.id
where
    o.id = $1
limit
    1;

-- name: AttachAskWallettoOrder :exec
update orders
set
    updated_at = now(),
    ask_wallet = $1
where
    id = $2;

-- name: AttachBidTransactionToOrder :exec
update orders
set
    updated_at = now(),
    bid_transaction = $1
where
    id = $2;

-- name: LookupPendingExecutions :many
select
    id
from
    executions
where
    exec_status = 'pending'
    and id > $1
order by
    id
limit
    $2;

-- name: UpdateExecutionOrdersStatuses :exec
update executions e
set
    updated_at = now(),
    bid_status = b.status,
    ask_status = a.status
from
    (
        select
            id,
            status
        from
            orders for no key
        update
    ) b,
    (
        select
            id,
            status
        from
            orders for no key
        update
    ) a
where
    e.id = $1
    and b.id = e.bid
    and a.id = e.ask;

-- name: FillOrderById :exec
update orders
set
    updated_at = now(),
    quantity = greatest(quantity - $1, 0),
    status = (
        case
            when (quantity - $1) <= 0 then 'filled'
            when (quantity - $1) > 0
            and (quantity - $1) < quantity then 'partially_filled'
            else status
        end
    )
where
    id = $2
    and status = 'submitted';

-- name: FindExecutionById :one
select
    *
from
    executions
where
    id = $1
limit
    1;

-- name: UpdateExecutionStatus :exec
update executions
set
    updated_at = now(),
    exec_status = $1
where
    id = $2;

-- name: LookupOpenOrders :many
select
    o.*,
    bw."type" as wallet_type
from
    orders o
    inner join wallet_transactions wt on o.bid_transaction = wt.id
    inner join wallets bw on wt.src = bw.id
where
    o.id > $1
    and o.status in ('pending', 'submitted', 'partially_filled')
order by
    o.id
limit
    $2;

-- name: MarkOrderSubmitted :exec
update orders
set
    status = 'submitted',
    updated_at = now()
where
    user_id = $1
    and id = $2;

-- name: MarkOrderRejected :exec
update orders
set
    status = 'rejected',
    updated_at = now()
where
    user_id = $1
    and id = $2;

-- name: FindOwnedOrder :one
select
    *
from
    orders
where
    user_id = $1
    and id = $2
limit
    1;

-- name: MarkOrderCancelled :exec
update orders
set
    status = 'cancelled',
    updated_at = now()
where
    user_id = $1
    and id = $2;

-- name: GetOrders :many
select
    *
from
    orders
where
    id > $1
order by
    id
limit
    $2;

-- name: FindOrderById :one
select
    *
from
    orders
where
    id = $1;

-- name: OrderExistsById :one
select
    exists (
        select
            1
        from
            orders
        where
            id = $1
    );

-- name: PlaceOrder :exec
INSERT INTO
    orders (
        id,
        user_id,
        symbol,
        side,
        order_type,
        quantity,
        limit_price,
        stop_price,
        client_order_id,
        tracing_id,
        currency,
        exchange_rate
    )
VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: CreateExecution :one
INSERT INTO
    "executions" (
        "id",
        "match_quantity",
        "symbol",
        "bid_price",
        "bid_status",
        "bid_currency",
        "bid_exchange_rate_snapshot",
        "exec_value",
        "ask_price",
        "ask_status",
        "ask_currency",
        "ask_exchange_rate_snapshot",
        "tracing_id",
        "user",
        "bid",
        "ask",
        "ask_wallet",
        "bid_wallet",
        "wallets_type"
    )
VALUES
    (
        $1, -- id
        $2, -- match_quantity
        $3, -- symbol
        $4, -- bid_price
        $5, -- bid_status
        $6, -- bid_currency
        $7, -- bid_exchange_rate_snapshot
        $8, -- exec_value
        $9, -- ask_price
        $10, -- ask_status
        $11, -- ask_currency
        $12, -- ask_exchange_rate_snapshot
        $13, -- tracing_id
        $14, -- user
        $15, -- bid
        $16, -- ask
        $17, -- ask_wallet
        $18, -- bid_wallet
        $19 -- wallets_type
    )
returning
    *;

-- name: ExecutionExists :one
select
    exists (
        select
            1
        from
            executions
        where
            ask = $1
            and bid = $2
    );