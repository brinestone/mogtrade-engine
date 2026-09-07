-- name: UpdateUserPrefs :exec
update "user"
set
    prefs = $1,
    updated_at = now()
where
    id = $2;