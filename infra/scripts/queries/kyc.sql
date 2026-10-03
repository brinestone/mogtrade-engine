-- name: UserHasActiveKYCProfile :one
select
    exists (
        select
            1
        from
            kyc_records
        where
            user_id = $1
            and (status in ('pending', 'verified', 'verifying'))
            and (created_at + verification_window > now())
    );

-- name: ListKycsByRiskProfile :many
select
    k.*,
    (k.created_at + k.valid_window)::timestamptz as expires_at,
    coalesce(
        k.status = 'verified'
        and now() <= (k.created_at + k.valid_window),
        false
    )::boolean as usable,
    (k.created_at + verification_window)::timestamptz as verify_before
from
    kyc_records k
where
    k.risk_profile = $1
order by
    verified_at desc nulls last;

-- name: ListKycs :many
select
    k.*,
    (k.created_at + k.valid_window)::timestamptz as expires_at,
    coalesce(
        k.status = 'verified'
        and now() <= (k.created_at + k.valid_window),
        false
    )::boolean as usable,
    (k.created_at + verification_window)::timestamptz as verify_before
from
    kyc_records k
order by
    created_at desc;

-- name: UpdateKycDocuments :exec
update kyc_records
set
    updated_at = now(),
    identity_doc = $2,
    proof_of_address = $3,
    status = 'pending',
    rejection_reason = null,
    verified_at = null,
    verified_by = null
where
    user_id = $1;

-- name: SetKycStatus :exec
update kyc_records
set
    updated_at = now(),
    status = $2
where
    user_id = $1;

-- name: MarkKycVerifying :exec
update kyc_records
set
    updated_at = now(),
    status = 'verifying'
where
    user_id = $1
    and status in ('pending', 'rejected', 'expired');

-- name: VerifyKyc :exec
update kyc_records
set
    updated_at = now(),
    status = 'verified',
    verified_at = now(),
    verified_by = $2,
    rejection_reason = null,
    risk_profile = $3
where
    user_id = $1
    and status not in ('expired', 'rejected');

-- name: RejectKyc :exec
update kyc_records
set
    updated_at = now(),
    status = 'rejected',
    verified_by = $2,
    rejection_reason = $3,
    verified_at = null
where
    user_id = $1
    and status not in ('expired', 'rejected');

-- name: LookupKycRecordByUser :one
select
    k.id,
    k.user_id,
    k.status,
    k.ready,
    (k.created_at + k.valid_window)::timestamptz as expires_at,
    coalesce(
        k.status = 'verified'
        and now() <= (k.created_at + k.valid_window),
        false
    )::boolean as usable,
    (k.created_at + verification_window)::timestamptz as verify_before
from
    kyc_records k
where
    k.user_id = $1
limit
    1;

-- name: LookupActiveKycForUser :one
select
    k.id,
    k.user_id,
    k.status,
    k.ready,
    (k.created_at + k.valid_window)::timestamptz as expires_at,
    coalesce(
        k.status = 'verified'
        and now() <= (k.created_at + k.valid_window),
        false
    )::boolean as usable,
    (k.created_at + verification_window)::timestamptz as verify_before
from
    kyc_records k
where
    k.user_id = $1
    and status not in ('expired', 'rejected')
limit
    1;

-- name: CreateKycRecord :exec
insert into
    kyc_records (id, valid_window, user_id, verification_window)
values
    ($1, $2, $3, $4);