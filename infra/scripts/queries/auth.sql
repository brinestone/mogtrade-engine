-- name: InvalidateRefreshTokensForDevice :exec
update refresh_tokens
set
    revoked_at = now()
where
    device_id = $1;

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
WHERE
    provider = 'credential'
    and account_id = $1
limit
    1;

-- name: FindUserByEmail :one
SELECT
    *
FROM
    "user"
WHERE
    email = $1
LIMIT
    1;

-- name: FindUserById :one
SELECT
    *
FROM
    "user"
WHERE
    id = $1
LIMIT
    1;

-- name: CreateUser :one
INSERT INTO
    "user" (id, name, email, image)
VALUES
    ($1, $2, $3, $4)
RETURNING
    created_at;

-- name: CreateCredentialAccount :one
INSERT INTO
    account (provider, id, account_id, user_id, "password")
VALUES
    ('credential', $1, $2, $3, $4)
RETURNING
    created_at;