-- name: ListBookings :many
SELECT
    b.id,
    b.customer_id,
    b.service_id,
    b.start_time,
    b.end_time,
    b.intentions,
    b.status,
    b.payment_status,
    b.stripe_checkout_session_id,
    b.stripe_payment_intent_id,
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
ORDER BY b.start_time DESC;

-- name: ListBookingsByCustomer :many
SELECT
    b.id,
    b.customer_id,
    b.service_id,
    b.start_time,
    b.end_time,
    b.intentions,
    b.status,
    b.payment_status,
    b.stripe_checkout_session_id,
    b.stripe_payment_intent_id,
    b.created_at,
    b.updated_at,
    s.name AS service_name,
    s.price AS service_price,
    s.minutes AS service_minutes
FROM bookings b
JOIN services s ON s.id = b.service_id
WHERE b.customer_id = ?
ORDER BY b.start_time DESC;

-- name: ListBookingsByStatus :many
SELECT
    b.id,
    b.customer_id,
    b.service_id,
    b.start_time,
    b.end_time,
    b.intentions,
    b.status,
    b.payment_status,
    b.created_at,
    c.name AS customer_name,
    c.email AS customer_email,
    s.name AS service_name
FROM bookings b
JOIN customers c ON c.id = b.customer_id
JOIN services s ON s.id = b.service_id
WHERE b.status = ?
ORDER BY b.start_time DESC;

-- name: GetBookingByID :one
SELECT
    b.id,
    b.customer_id,
    b.service_id,
    b.start_time,
    b.end_time,
    b.intentions,
    b.status,
    b.payment_status,
    b.stripe_checkout_session_id,
    b.stripe_payment_intent_id,
    b.created_at,
    b.updated_at,
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
INSERT INTO bookings (customer_id, service_id, start_time, end_time, intentions, status, payment_status)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateBooking :one
UPDATE bookings
SET customer_id = ?, service_id = ?, start_time = ?, end_time = ?, intentions = ?, status = ?, payment_status = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?
RETURNING *;

-- name: UpdateBookingStatus :exec
UPDATE bookings
SET status = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?;

-- name: UpdateBookingPayment :exec
UPDATE bookings
SET payment_status = ?,
    stripe_payment_intent_id = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?;

-- name: SetBookingCheckoutSession :exec
UPDATE bookings
SET stripe_checkout_session_id = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?;

-- name: ConfirmBookingByCheckoutSession :exec
UPDATE bookings
SET status = 'confirmed',
    payment_status = 'paid',
    stripe_payment_intent_id = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE stripe_checkout_session_id = ?;

-- name: CancelBookingByCheckoutSession :exec
UPDATE bookings
SET status = 'cancelled',
    payment_status = 'failed',
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE stripe_checkout_session_id = ?
  AND payment_status != 'paid';

-- name: DeleteBooking :exec
DELETE FROM bookings WHERE id = ?;

-- name: ListBookingsByDateRange :many
SELECT * FROM bookings
WHERE start_time >= sqlc.arg(day_start) AND start_time < sqlc.arg(day_end)
  AND status IN ('pending', 'confirmed')
ORDER BY start_time ASC;

-- name: CountOverlappingBookings :one
SELECT count(*) FROM bookings
WHERE status IN ('pending', 'confirmed')
  AND start_time < sqlc.arg(end_time)
  AND end_time > sqlc.arg(start_time);
