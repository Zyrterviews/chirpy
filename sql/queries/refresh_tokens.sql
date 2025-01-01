-- name: CreateRefreshToken :one
INSERT INTO
    refresh_tokens (token, user_id, expires_at)
VALUES
    ($1, $2, $3)
RETURNING
    *;

-- name: GetRefreshToken :one
SELECT
    *
FROM
    refresh_tokens
WHERE
    token = $1;

-- name: GetUserFromRefreshToken :one
SELECT
    *
FROM
    users
    INNER JOIN refresh_tokens ON users.id = refresh_tokens.user_id
WHERE
    refresh_tokens.token = $1;

-- name: RevokeRefreshToken :exec
UPDATE
    refresh_tokens
SET
    updated_at = (NOW() AT TIME ZONE 'utc'),
    revoked_at = (NOW() AT TIME ZONE 'utc')
WHERE
    token = $1;
