alter table orders
drop column average_fill_price;

alter table orders
drop column filled_quantity;

alter table orders
add column fee numeric(18, 4) not null default 0.0;

alter table orders
alter column fee
drop default;

alter table orders
add column currency text not null default '';

alter table orders
alter column currency
drop default;

alter table orders
add column exchange_rate numeric(18, 4) not null default 1.0;

alter table orders
alter column exchange_rate
drop default;