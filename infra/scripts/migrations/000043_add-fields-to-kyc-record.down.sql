alter table kyc_records
drop column address;

alter table kyc_records
drop column dob;

alter table kyc_records
drop column gender;

alter table kyc_records
drop column legal_names;

drop type gender;