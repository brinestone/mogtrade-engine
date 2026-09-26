alter table orders
drop column exchange_rate;

alter table orders
drop column currency;

alter table orders
drop column fee;

alter table orders
add column filled_quantity numeric(18, 8) default 0.0;

alter table orders
add column average_fill_price numeric(18, 8);