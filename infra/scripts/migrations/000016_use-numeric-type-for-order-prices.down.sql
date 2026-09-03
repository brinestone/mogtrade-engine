ALTER TABLE orders
ALTER COLUMN limit_price
TYPE float using limit_price::float;

ALTER TABLE orders
ALTER COLUMN quantity
TYPE float using quantity::float;

alter table orders
alter column stop_price
type float using stop_price::float;

alter table orders
alter column filled_quantity
type float using filled_quantity::float;

alter table orders
alter column average_fill_price
type float using average_fill_price::float;