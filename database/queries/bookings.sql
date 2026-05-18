-- name: GetBookingByCheckoutID :one
SELECT * FROM bookings
WHERE stripe_checkout_session_id = $1;

-- name: CreateBooking :one
INSERT INTO bookings (
    customer_id, service_id, starts_at, ends_at,
    status, service_name, service_price,
    customer_notes, stripe_checkout_session_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetOverlappingBookings :many
SELECT * FROM bookings
WHERE status NOT IN ('cancelled')
  AND starts_at < @ends_at
  AND ends_at > @starts_at;

-- name: UpdateBookingToPaid :one
UPDATE bookings
SET status = 'paid',
    stripe_payment_intent_id = $2,
    paid_at = now(),
    updated_at = now()
WHERE stripe_checkout_session_id = $1
RETURNING *;

-- name: UpdateBookingToFailed :one
UPDATE bookings
SET status = 'payment_failed',
    updated_at = now()
WHERE stripe_payment_intent_id = $1
RETURNING *;
