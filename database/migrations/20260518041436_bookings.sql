-- +goose Up
CREATE TABLE bookings (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
customer_id UUID NOT NULL,
FOREIGN KEY (customer) REFERENCES users(id) ON DELETE CASCADE,
service_id UUID NOT NULL,
FOREIGN KEY (service) REFERENCES services(id) ON DELETE CASCADE,
starts_at TIMESTAMP NOT NULL,
ends_at TIMESTAMP NOT NULL,
status TEXT NOT NULL DEFAULT 'pending',
service_name TEXT NOT NULL,
customer_notes TEXT,
stripe_checkout_session_id TEXT UNIQUE,
stripe_payment_intent_id TEXT UNIQUE,
paid_at TIMESTAMP,
cancelled_at TIMESTAMP,
created_at TIMESTAMP NOT NULL DEFAULT now(),
updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE bookings;
