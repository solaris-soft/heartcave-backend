-- name: CreateService :one
INSERT INTO services (name, price, description, session_minutes)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetServiceByName :one
SELECT * FROM services
WHERE name = $1;

-- name: GetAllServices :many
SELECT * FROM services;
