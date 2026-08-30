-- name: GetUserById :one
SELECT
    *
FROM
    user
WHERE
    id = @user_id
LIMIT
    1;