-- name: CreateAvailability :one
INSERT INTO service_availability (service_id, day_of_week, start_time, end_time)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetAvailabilityByID :one
SELECT * FROM service_availability
WHERE id = $1;

-- name: GetAllAvailability :many
SELECT * FROM service_availability;

-- name: GetAvailabilityByServiceID :many
SELECT * FROM service_availability
WHERE service_id = $1;

-- name: UpdateAvailabilityByID :one
UPDATE service_availability
SET service_id = $1, day_of_week = $2, start_time = $3, end_time = $4, updated_at = now()
WHERE id = $5
RETURNING *;

-- name: DeleteAvailabilityByID :exec
DELETE FROM service_availability
WHERE id = $1;
