create type transaction_status as enum(
    'processing',
    'failed',
    'cancelled',
    'completed',
    'retrying'
);

alter table wallet_transactions
add column status transaction_status not null default 'processing';