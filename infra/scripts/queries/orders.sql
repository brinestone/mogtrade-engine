-- name: PlaceOrder :one
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
        average_fill_price
    )
VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING
    id;

-- name: ApplyOrderFill :exec
UPDATE orders
SET
    filled_quantity = filled_quantity + $1,
    status = (
        CASE
            WHEN filled_quantity + $1 >= quantity THEN 'filled'
            ELSE 'partially_filled'
        END
    )::order_status,
    updated_at = now()
WHERE
    id = $2
    AND status IN ('submitted', 'partially_filled');