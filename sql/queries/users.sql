-- name: CreateUser :one
INSERT INTO
    users (email, hashed_password)
VALUES
    ($1, $2)
RETURNING
    *;

-- name: GetUserByEmail :one
SELECT
    *
FROM
    users
WHERE
    email = $1;

-- name: GetUserByID :one
SELECT
    *
FROM
    users
WHERE
    id = $1;

-- name: UpdateUser :one
UPDATE
    users
SET
    updated_at = (NOW() AT TIME ZONE 'utc'),
    email = $1,
    hashed_password = $2
WHERE
    id = $3
RETURNING
    *;

-- name: DeleteAllUsers :exec
DELETE FROM
    users;

-- name: SetUserAsChirpyRed :one
UPDATE
    users
SET
    updated_at = (NOW() AT TIME ZONE 'utc'),
    is_chirpy_red = TRUE
WHERE
    id = $1
RETURNING
    *;
