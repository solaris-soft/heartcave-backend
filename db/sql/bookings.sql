-- name: ListBookings :many
SELECT
    b.id,
    b.customer_id,
    b.service_id,
    b.date,
    b.intentions,
    b.status,
    b.created_at,
    b.updated_at,
    c.name AS customer_name,
    c.email AS customer_email,
    c.phone AS customer_phone,
    s.name AS service_name,
    s.price AS service_price,
    s.minutes AS service_minutes
FROM bookings b
JOIN customers c ON c.id = b.customer_id
JOIN services s ON s.id = b.service_id
ORDER BY b.date DESC;

-- name: ListBookingsByCustomer :many
SELECT
    b.*,
    s.name AS service_name,
    s.price AS service_price
FROM bookings b
JOIN services s ON s.id = b.service_id
WHERE b.customer_id = ?
ORDER BY b.date DESC;

-- name: ListBookingsByStatus :many
SELECT
    b.id,
    b.customer_id,
    b.service_id,
    b.date,
    b.intentions,
    b.status,
    b.created_at,
    c.name AS customer_name,
    c.email AS customer_email,
    s.name AS service_name
FROM bookings b
JOIN customers c ON c.id = b.customer_id
JOIN services s ON s.id = b.service_id
WHERE b.status = ?
ORDER BY b.date DESC;

-- name: GetBookingByID :one
SELECT
    b.*,
    c.name AS customer_name,
    c.email AS customer_email,
    c.phone AS customer_phone,
    s.name AS service_name,
    s.description AS service_description,
    s.price AS service_price,
    s.minutes AS service_minutes
FROM bookings b
JOIN customers c ON c.id = b.customer_id
JOIN services s ON s.id = b.service_id
WHERE b.id = ?
LIMIT 1;

-- name: CreateBooking :one
INSERT INTO bookings (customer_id, service_id, date, intentions, status)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateBooking :one
UPDATE bookings
SET customer_id = ?, service_id = ?, date = ?, intentions = ?, status = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?
RETURNING *;

-- name: UpdateBookingStatus :exec
UPDATE bookings
SET status = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?;

-- name: DeleteBooking :exec
DELETE FROM bookings WHERE id = ?;

-- name: ListBookingsByDate :many
SELECT * FROM bookings WHERE date = ? ORDER BY date ASC;
