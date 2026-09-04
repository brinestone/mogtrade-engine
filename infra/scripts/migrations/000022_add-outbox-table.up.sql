-- 1. Define the status enum for outbox messages
create type outbox_status as enum('pending', 'processing', 'published', 'failed');

-- 2. Refactored Outbox Table
create table
    outbox (
        id varchar(26) primary key,
        tracing_id varchar(26) not null,
        topic text not null,
        payload jsonb not null,
        status outbox_status not null default 'pending',
        retry_count int not null default 0,
        max_retries int not null default 5,
        error_message text,
        available_at timestamptz not null default now(), -- Used for exponential backoff delays on failure
        recorded_at timestamptz not null default now(),
        updated_at timestamptz not null default now()
    );

-- Crucial index for the publisher polling query
create index outbox_polling_idx on outbox (status, available_at)
where
    status in ('pending', 'processing');

create index "outbox_tracing_id_idx" on "outbox" ("tracing_id");

create table
    processed_events (
        id varchar(26) primary key,
        outbox_id varchar(26) not null,
        service text not null,
        succeeded_at timestamptz not null default now(),
        constraint processed_events_unique_delivery unique (outbox_id, service)
    );

create index "processed_events_service_idx" on "processed_events" ("service");

alter table processed_events
add constraint "processed_events_outbox_id_fk" foreign key ("outbox_id") references "outbox" ("id") on delete cascade;