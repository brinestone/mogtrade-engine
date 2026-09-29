create type gender as enum('male', 'female', 'undefined');

alter table kyc_records
add column legal_names text not null default '';

alter table kyc_records
alter column legal_names
drop default;

alter table kyc_records
add column gender gender not null default 'undefined';

alter table kyc_records
alter column gender
drop default;

alter table kyc_records
add column dob date not null default (now() - '18 years'::interval)::date;

alter table kyc_records
alter column dob
drop default;

alter table kyc_records
add column address jsonb;