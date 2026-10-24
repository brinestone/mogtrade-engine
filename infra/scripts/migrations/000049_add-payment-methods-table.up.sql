create table
    payment_methods (
        id varchar(26) not null primary key,
        "owner" varchar(26) not null,
        name text not null,
        account_params jsonb not null default '{}'::jsonb,
        enabled boolean default true,
        created_at timestamptz not null default now(),
        updated_at timestamptz not null default now(),
        default_currency text not null,
        is_default boolean default false
    );

alter table payment_methods
add constraint "payment_methods_owner_user_id_fk" foreign key ("owner") references "user" ("id") on delete cascade;

create unique index "payment_methods_owner_name_uidx" on "payment_methods" ("owner", "name");

create index "payment_methods_name_idx" on "payment_methods" ("name");

create index "payment_methods_owner_idx" on "payment_methods" ("owner");

create index "payment_methods_default_currency_idx" on "payment_methods" ("default_currency");