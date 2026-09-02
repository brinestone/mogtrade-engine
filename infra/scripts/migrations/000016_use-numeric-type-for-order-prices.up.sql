ALTER TABLE orders
ALTER COLUMN limit_price
TYPE NUMERIC(18, 8);

ALTER TABLE orders
ALTER COLUMN quantity
TYPE NUMERIC(18, 8);

alter table orders
alter column stop_price
type numeric(18, 8);

alter table orders
alter column filled_quantity
type numeric(18, 8);

alter table orders
alter column average_fill_price
type numeric(18, 8);