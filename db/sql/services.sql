-- name: ListServices :many
SELECT * FROM services ORDER BY name;

-- name: GetServiceByID :one
SELECT * FROM services WHERE id = ? LIMIT 1;

-- name: GetServiceByName :one
SELECT * FROM services WHERE name = ? LIMIT 1;

-- name: CreateService :one
INSERT INTO services (name, description, price, minutes)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: UpdateService :one
UPDATE services
SET name = ?, description = ?, price = ?, minutes = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?
RETURNING *;

-- name: DeleteService :exec
DELETE FROM services WHERE id = ?;
