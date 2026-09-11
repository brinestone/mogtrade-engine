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
    'retrying'
);

alter table wallet_transactions
alter column status
type transaction_status using status::transaction_status;

alter table wallet_transactions
alter column status set default 'processing'::transaction_status;