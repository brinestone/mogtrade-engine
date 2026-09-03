-- name: CreateRefreshToken :exec
insert into
    refresh_tokens (id, user_id, token_hash, valid_window)
values
    ($1, $2, $3, $4);

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
    *;

-- name: CreateCredentialAccount :one
INSERT INTO
    account (id, account_id, provider, user_id, "password")
VALUES
    ($1, $2, $3, $4, $5)
RETURNING
    *;