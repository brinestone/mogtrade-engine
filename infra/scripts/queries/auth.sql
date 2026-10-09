-- name: SetUserCredentialPassword :exec
update account
set
    "password" = $1,
    updated_at = now()
where
    provider = 'credential'
    and user_id = $2;

-- name: UserExistsWithId :one
select
    exists (
        select
            1
        from
            "user"
        where
            id = $1
    );

-- name: VerifyUserEmail :exec
update "user"
set
    email_verified = true,
    updated_at = now()
where
    id = $1;

-- name: IsEmailAvailable :one
select
    not exists (
        select
            1
        from
            "user"
        where
            email = $1
    );

-- name: RemoveStaleRefreshTokens :exec
delete from refresh_tokens
where
    revoked_at is not null
    or (created_at + valid_window) < now();

-- name: LookupRefreshTokenByDevice :one
select
    rts.user_id,
    rts.usable
from
    refresh_token_states rts
where
    rts.device_id = $1
    and token_hash = $2
limit
    1;

-- name: InvalidateRefreshTokensForDevice :exec
update refresh_tokens
set
    revoked_at = now()
where
    device_id = $1
    and revoked_at is null;

-- name: CredentialAccountExistsByIdentifier :one
select
    exists (
        select
            1
        from
            account
        where
            provider = 'credential'
            and account_id = $1
    );

-- name: CreateRefreshToken :exec
insert into
    refresh_tokens (id, user_id, token_hash, valid_window, device_id)
values
    ($1, $2, $3, $4, $5);

-- name: FindCredentialAccountById :one
SELECT
    *
FROM
    account
where
    provider = 'credential'
    and account_id = $1
limit
    1;

-- name: FindUserByEmail :one
SELECT
    *
FROM
    "user"
where
    email = $1
limit
    1;

-- name: FindUserById :one
SELECT
    *
FROM
    "user"
where
    id = $1
limit
    1;

-- name: CreateUser :one
INSERT INTO
    "user" (id, name, email, image)
values
    ($1, $2, $3, $4)
RETURNING
    created_at;

-- name: CreateCredentialAccount :one
INSERT INTO
    account (provider, id, account_id, user_id, "password")
values
    ('credential', $1, $2, $3, $4)
RETURNING
    created_at;

-- name: CreateVerification :exec
insert into
    verifications (
        id,
        user_id,
        token,
        valid_window,
        "type",
        ip,
        callback_url
    )
values
    ($1, $2, $3, $4, $5, $6, $7);

-- name: GetPrimedVerificationByToken :one
select
    v.*,
    (created_at + valid_window)::timestamptz as expires_at
from
    verifications v
where
    v.token = $1
    and v.status = 'primed'
    and v.used_at is null
    and (v.created_at + v.valid_window) <= now()
limit
    1;

-- name: GetUnusedVerificationByToken :one
select
    v.*,
    (created_at + valid_window)::timestamptz as expires_at
from
    verifications v
where
    v.token = $1
    and v.used = false
    and (v.created_at + v.valid_window) <= now()
limit
    1;

-- name: PrimeVerification :exec
update verifications
set
    status = 'primed'
where
    token = $1;

-- name: MarkVerificationUsed :exec
update verifications
set
    used_at = now()
where
    token = $1;

-- name: ListUnusedVerificationsByUser :many
select
    id,
    token,
    "type",
    (created_at + valid_window)::timestamptz as expires_at
from
    verifications
where
    user_id = $1
    and used = false;

-- name: UnusedVerificationExistsByToken :one
select
    exists (
        select
            1
        from
            verifications
        where
            token = $1
            and not used
            and ((created_at + valid_window) < now())
    );