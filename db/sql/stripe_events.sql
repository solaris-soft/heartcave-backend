-- name: RecordStripeEvent :one
INSERT OR IGNORE INTO stripe_events (id, event_type)
VALUES (?, ?)
RETURNING id;

-- name: MarkStripeEventProcessed :exec
UPDATE stripe_events
SET processed_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?;

-- name: DeleteStripeEvent :exec
DELETE FROM stripe_events WHERE id = ?;
