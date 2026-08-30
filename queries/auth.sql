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
    account (
        id,
        issuer,
        account_id,
        provider,
        user_id,
        "password"
    )
VALUES
    ($1, $2, $3, $4, $5, $6)
RETURNING
    *;