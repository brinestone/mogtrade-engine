CREATE TYPE order_status AS ENUM(
    'pending',
    'submitted',
    'partially_filled',
    'filled',
    'cancelled',
    'rejected',
    'expired'
);