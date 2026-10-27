drop table executions cascade;

drop type execution_status;

create type execution_status as enum('pending', 'failed', 'filled');

create table
    executions (
        id varchar(26) not null primary key,
        match_quantity numeric(18, 4) not null,
        symbol text not null,
        "user" varchar(26),
        bid varchar(26),
        bid_price numeric(18, 4) not null,
        bid_status order_status not null,
        bid_currency text not null,
        bid_exchange_rate_snapshot numeric(18, 4) not null,
        exec_value numeric(18, 4) not null,
        ask varchar(26),
        ask_price numeric(18, 4) not null,
        ask_status order_status not null,
        ask_currency text not null,
        ask_exchange_rate_snapshot numeric(18, 4) not null,
        exec_status execution_status not null default 'pending',
        recorded_at timestamptz not null default now(),
        updated_at timestamptz not null default now(),
        tracing_id text not null,
        ask_wallet varchar(26),
        bid_wallet varchar(26),
        wallets_type wallet_type not null
    );
alter table executions
add constraint "executions_bid_wallet_wallets_id_fk" foreign key ("bid_wallet") references wallets ("id") on delete set null;

alter table executions
add constraint "executions_ask_wallet_wallets_id_fk" foreign key ("ask_wallet") references wallets ("id") on delete set null;

create index "executions_bid_wallet_idx" on "executions" ("bid_wallet");

create index "executions_ask_wallet_idx" on "executions" ("ask_wallet")
where
    "ask_wallet" is not null;

alter table executions
add constraint "executions_ask_orders_id_fk" foreign key ("ask") references "orders" ("id") on delete set null;

alter table executions
add constraint "executions_bid_orders_id_fk" foreign key ("bid") references "orders" ("id") on delete set null;

alter table executions
add constraint "executions_user_user_id_fk" foreign key ("user") references "user" ("id") on delete set null;

create index if not exists "executions_recorded_at_idx" on "executions" ("tracing_id");

create index if not exists "executions_recorded_at_idx" on "executions" ("recorded_at");

create index if not exists "executions_recorded_at_idx" on "executions" ("updated_at");

create unique index if not exists "executions_ask_bid_uidx" on "executions" ("ask", "bid")
where
    "ask" is not null
    and "bid" is not null;

create index if not exists "executions_ask_idx" on "executions" ("ask")
where
    ask is not null;

create index if not exists "executions_bid_idx" on "executions" ("bid")
where
    bid is not null;

create index if not exists "executions_user_idx" on "executions" ("user")
where
    "user" is not null;

create index if not exists "executions_symbol_idx" on "executions" ("symbol");