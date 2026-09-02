ALTER TABLE orders
ALTER COLUMN limit_price
TYPE float;

ALTER TABLE orders
ALTER COLUMN quantity
TYPE float;

alter table orders
alter column stop_price
type float;

alter table orders
alter column filled_quantity
type float;

alter table orders
alter column average_fill_price
type float;