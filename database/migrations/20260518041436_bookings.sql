-- +goose Up
CREATE TABLE bookings (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
customer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
service_id UUID NOT NULL REFERENCES services(id) ON DELETE RESTRICT,
starts_at TIMESTAMP NOT NULL,
ends_at TIMESTAMP NOT NULL,
status TEXT NOT NULL DEFAULT 'pending',
service_name TEXT NOT NULL,
service_price NUMERIC(19, 2) NOT NULL,
customer_notes TEXT,
stripe_checkout_session_id TEXT UNIQUE,
stripe_payment_intent_id TEXT UNIQUE,
paid_at TIMESTAMP,
cancelled_at TIMESTAMP,
created_at TIMESTAMP NOT NULL DEFAULT now(),
updated_at TIMESTAMP NOT NULL DEFAULT now(),
CHECK (ends_at > starts_at),
CHECK (
    status IN (
    'pending',
    'paid',
    'cancelled',
    'completed',
    'payment_failed',
    'refunded'
    )
    )
);

-- +goose Down
DROP TABLE bookings;
