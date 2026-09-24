-- name: ListKycsByRiskProfile :many
select
    k.*,
    (k.created_at + k.valid_window)::timestamptz as expires_at,
    (k.created_at + k.valid_window) <= now()
    or k.status not in ('rejected', 'expired')::boolean as usable
from
    kyc_records k
where
    k.risk_profile = $1
order by
    verified_at desc;

-- name: ListKycs :many
select
    k.*,
    (k.created_at + k.valid_window) <= now()
    or k.status not in ('rejected', 'expired')::boolean as usable
from
    kyc_records k;

-- name: UpdateKycDocuments :exec
update kyc_records
set
    updated_at = now(),
    identity_doc = $1,
    proof_of_address = $2
where
    user_id = $1;

-- name: SetKycStatus :exec
update kyc_records
set
    updated_at = now(),
    status = $2
where
    user_id = $1;

-- name: GetKycRecordByUser :one
select
    *
from
    kyc_records
where
    user_id = $1
limit
    1;

-- name: CreateKycRecord :exec
insert into
    kyc_records (
        id,
        valid_window,
        identity_doc,
        proof_of_address,
        user_id
    )
values
    ($1, $2, $3, $4, $5);