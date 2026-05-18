-- name: GetStripeEventByID :one
SELECT * FROM stripe_webhook_events
WHERE id = $1;

-- name: CreateProcessedStripeEvent :exec
INSERT INTO stripe_webhook_events (id)
VALUES ($1);
