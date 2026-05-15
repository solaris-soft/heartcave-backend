-- +goose Up
UPDATE customers
SET email = email || '+duplicate-' || id
WHERE id NOT IN (
    SELECT min(id)
    FROM customers
    GROUP BY lower(email)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_email ON customers(email);

CREATE TABLE bookings_new (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id                INTEGER NOT NULL REFERENCES customers(id),
    service_id                 INTEGER NOT NULL REFERENCES services(id),
    start_time                 TEXT    NOT NULL,
    end_time                   TEXT    NOT NULL,
    intentions                 TEXT    NOT NULL DEFAULT '',
    status                     TEXT    NOT NULL DEFAULT 'pending',
    payment_status             TEXT    NOT NULL DEFAULT 'unpaid',
    stripe_checkout_session_id TEXT    NOT NULL DEFAULT '',
    stripe_payment_intent_id   TEXT    NOT NULL DEFAULT '',
    created_at                 TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at                 TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

INSERT INTO bookings_new (
    id,
    customer_id,
    service_id,
    start_time,
    end_time,
    intentions,
    status,
    created_at,
    updated_at
)
SELECT
    id,
    customer_id,
    service_id,
    CASE
        WHEN instr(date, 'T') > 0 THEN date
        WHEN length(date) = 16 THEN replace(date, ' ', 'T') || ':00'
        WHEN length(date) = 10 THEN date || 'T00:00:00'
        ELSE date
    END,
    CASE
        WHEN instr(date, 'T') > 0 THEN datetime(date, '+60 minutes')
        WHEN length(date) = 16 THEN replace(datetime(replace(date, ' ', 'T'), '+60 minutes'), ' ', 'T')
        WHEN length(date) = 10 THEN date || 'T01:00:00'
        ELSE date
    END,
    intentions,
    status,
    created_at,
    updated_at
FROM bookings;

DROP TABLE bookings;
ALTER TABLE bookings_new RENAME TO bookings;

CREATE INDEX IF NOT EXISTS idx_bookings_customer_id ON bookings(customer_id);
CREATE INDEX IF NOT EXISTS idx_bookings_start_time ON bookings(start_time);
CREATE UNIQUE INDEX IF NOT EXISTS idx_bookings_stripe_checkout_session_id
    ON bookings(stripe_checkout_session_id)
    WHERE stripe_checkout_session_id != '';

CREATE TABLE IF NOT EXISTS stripe_events (
    id           TEXT PRIMARY KEY,
    event_type   TEXT NOT NULL,
    processed_at TEXT,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- +goose Down
DROP TABLE IF EXISTS stripe_events;

CREATE TABLE bookings_old (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id INTEGER NOT NULL REFERENCES customers(id),
    service_id  INTEGER NOT NULL REFERENCES services(id),
    date        TEXT    NOT NULL,
    intentions  TEXT    NOT NULL,
    status      TEXT    NOT NULL DEFAULT 'pending',
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

INSERT INTO bookings_old (
    id,
    customer_id,
    service_id,
    date,
    intentions,
    status,
    created_at,
    updated_at
)
SELECT
    id,
    customer_id,
    service_id,
    start_time,
    intentions,
    status,
    created_at,
    updated_at
FROM bookings;

DROP TABLE bookings;
ALTER TABLE bookings_old RENAME TO bookings;
