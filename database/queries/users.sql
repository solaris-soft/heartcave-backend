-- name: CreateUser :one
INSERT INTO users (name, email, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: UpdateUserByEmail :exec
UPDATE users
SET password_hash = $1
WHERE email = $2;

-- name: UpdateUserByID :one
UPDATE users
SET email = COALESCE(sqlc.narg(email), email),
    password_hash = COALESCE(sqlc.narg(password_hash), password_hash)
WHERE id = @id
RETURNING *;
