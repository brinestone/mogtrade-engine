alter table "kyc_records"
drop constraint "kyc_records_verified_by_user_id_fk";

alter table "kyc_records"
drop constraint "kyc_records_user_id_user_id_fk";

drop index "kyc_records_verified_at_idx";

drop index "kyc_records_user_id_uidx";

drop table kyc_records;

drop type kyc_risk_profile;

drop type kyc_status;