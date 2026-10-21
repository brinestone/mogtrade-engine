-- name: FindOpenOrders :many
select
    *
from
    orders
where
    id > $1
    and status in ('pending', 'submitted', 'partially_filled')
order by
    id
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

-- name: CreateExecution :exec
insert into
    executions (
        id,
        "order",
        price,
        quantity,
        order_side,
        order_status,
        "order_type",
        symbol,
        "user",
        fee_currency,
        fee_rate,
        fee_amount,
        tracing_id,
        exec_status
    )
values
    (
        $1,
        $2,
        $3,
        $4,
        $5,
        $6,
        $7,
        $8,
        $9,
        $10,
        $11,
        $12,
        $13,
        $14
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
        fee,
        currency,
        exchange_rate
    )
VALUES
    (
        $1,
        $2,
        $3,
        $4,
        $5,
        $6,
        $7,
        $8,
        $9,
        $10,
        $11,
        $12,
        $13
    );