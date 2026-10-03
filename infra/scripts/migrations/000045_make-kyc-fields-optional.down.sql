alter table kyc_records
drop column ready;

alter table kyc_records
alter column dob
set default (now() - '18 years'::interval)::date;

alter table kyc_records
alter column dob
set not null;

alter table kyc_records
alter column dob
drop default;

alter table kyc_records
alter column gender
set default 'undefined';

alter table kyc_records
alter column gender
set not null;

alter table kyc_records
alter column gender
drop default;

alter table kyc_records
alter column legal_names
set default 'N/A';

alter table kyc_records
alter column legal_names
set not null;

alter table kyc_records
alter column legal_names
drop default;