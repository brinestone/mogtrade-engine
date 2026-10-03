alter table kyc_records
add column verification_window interval not null default '24 hours';

alter table kyc_records
alter column verification_window
drop default;