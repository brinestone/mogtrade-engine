create type kyc_status as enum(
    'pending',
    'rejected',
    'verified',
    'expired',
    'verifying'
);

create type kyc_risk_profile as enum('low', 'medium', 'high');

CREATE TABLE
    kyc_records (
        id varchar(26) PRIMARY KEY,
        user_id varchar(26),
        status kyc_status NOT NULL DEFAULT 'pending',
        risk_profile kyc_risk_profile NOT NULL DEFAULT 'low',
        verified_at timestamptz,
        valid_window interval NOT NULL,
        identity_doc jsonb NOT NULL DEFAULT '{"front_hash": "", "back_hash": "", "selfie_hash": "", "uri": ""}'::jsonb,
        proof_of_address jsonb NOT NULL DEFAULT '{"doc_hash": "", "uri": ""}'::jsonb,
        created_at timestamptz DEFAULT now() NOT NULL,
        updated_at timestamptz DEFAULT now() NOT NULL,
        verified_by varchar(26),
        rejection_reason text
    );

create unique index "kyc_records_user_id_uidx" on "kyc_records" ("user_id")
where
    user_id is not null;

create index "kyc_records_verified_at_idx" on "kyc_records" ("verified_at")
where
    "verified_at" is not null;

alter table kyc_records
add constraint "kyc_records_user_id_user_id_fk" foreign key ("user_id") references "user" ("id") on delete set null;

alter table kyc_records
add constraint "kyc_records_verified_by_user_id_fk" foreign key ("verified_by") references "user" ("id") on delete set null;