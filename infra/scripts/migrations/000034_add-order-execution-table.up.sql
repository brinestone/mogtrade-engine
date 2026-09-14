create type execution_status as enum('failed', 'filled');

create table
    executions (
        id varchar(26) not null primary key,
        "order" varchar(26) not null,
        price numeric(18, 4) not null,
        quantity numeric(18, 4) not null,
        side order_side not null,
        status order_status not null,
        "type" order_type not null,
        symbol text not null,
        "user" varchar(26),
        fee_currency text not null,
        fee_rate numeric(18, 4) not null,
        fee_amount numeric(18, 4) not null,
        exec_status execution_status not null,
        recorded_at timestamptz not null default now()
    );

create index "executions_order_idx" on "executions" ("order");

create index "executions_user_idx" on "executions" ("user")
where
    "user" is not null;

create index "executions_symbol_idx" on "executions" ("symbol");

create index "executions_fee_currency_idx" on "executions" ("fee_currency");

alter table executions
add constraint "executions_user_user_id_fk" foreign key ("user") references "user" (id) on delete set null;

alter table executions
add constraint "executions_order_orders_id_fk" foreign key ("order") references orders (id) on delete cascade;