-- name: CreateService :one
INSERT INTO services (name, price, description, session_minutes)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetServiceByName :one
SELECT * FROM services
WHERE name = $1;

-- name: GetAllServices :many
SELECT * FROM services;

-- name: GetServiceByID :one
SELECT * FROM services
WHERE id = $1;

-- name: UpdateServiceByID :one
UPDATE services
SET name = $1, price = $2, description = $3, session_minutes = $4
WHERE id = $5
RETURNING *;

-- name: DeleteServiceByID :exec
DELETE FROM services
WHERE id = $1;
