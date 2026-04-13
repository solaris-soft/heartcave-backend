-- name: ListSchedule :many
SELECT * FROM admin_schedule ORDER BY day_of_week ASC, start_time ASC;

-- name: GetScheduleByDay :many
SELECT * FROM admin_schedule WHERE day_of_week = ? ORDER BY start_time ASC;

-- name: CreateScheduleEntry :one
INSERT INTO admin_schedule (day_of_week, start_time, end_time, slot_minutes)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: DeleteScheduleEntry :exec
DELETE FROM admin_schedule WHERE id = ?;
