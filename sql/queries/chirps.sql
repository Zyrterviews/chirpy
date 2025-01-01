-- name: CreateChirp :one
INSERT INTO
    chirps (body, user_id)
VALUES
    ($1, $2)
RETURNING
    *;

-- name: GetChirpByID :one
SELECT
    *
FROM
    chirps
WHERE
    id = $1;

-- name: GetAllChirpsForUser :many
SELECT
    *
FROM
    chirps
WHERE
    user_id = $1
ORDER BY
    CASE
        WHEN $2 LIKE 'asc' THEN created_at
    END ASC,
    CASE
        WHEN $2 LIKE 'desc' THEN created_at
    END DESC,
    CASE
        WHEN $2 NOT LIKE 'asc'
        AND $2 NOT LIKE 'desc' THEN created_at
    END ASC;

-- name: GetAllChirps :many
SELECT
    *
FROM
    chirps
ORDER BY
    CASE
        WHEN $1 LIKE 'asc' THEN created_at
    END ASC,
    CASE
        WHEN $1 LIKE 'desc' THEN created_at
    END DESC,
    CASE
        WHEN $1 NOT LIKE 'asc'
        AND $1 NOT LIKE 'desc' THEN created_at
    END ASC;

-- name: DeleteChirpByID :exec
DELETE FROM
    chirps
WHERE
    id = $1;
