alter table kyc_records
alter column legal_names
drop not null;

alter table kyc_records
alter column gender
drop not null;

alter table kyc_records
alter column dob
drop not null;

alter table kyc_records
add column ready boolean generated always as (
    legal_names is not null
    and gender is not null
    and dob is not null
) stored;