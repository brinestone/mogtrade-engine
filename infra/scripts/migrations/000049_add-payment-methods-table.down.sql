drop index "payment_methods_default_currency_idx";

drop index "payment_methods_owner_idx";

drop index "payment_methods_name_idx";

drop index "payment_methods_owner_name_uidx";

alter table payment_methods
drop constraint "payment_methods_owner_user_id_fk";

drop table "payment_methods";