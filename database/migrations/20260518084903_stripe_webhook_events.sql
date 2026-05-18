-- +goose Up
CREATE TABLE stripe_webhook_events (
id TEXT PRIMARY KEY,
processed_at TIMESTAMP NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE stripe_webhook_events;
