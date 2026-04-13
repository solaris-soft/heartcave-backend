-- +goose Up
ALTER TABLE customers ADD COLUMN password_hash TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS admin_schedule (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    day_of_week  INTEGER NOT NULL,
    start_time   TEXT    NOT NULL,
    end_time     TEXT    NOT NULL,
    slot_minutes INTEGER NOT NULL DEFAULT 60,
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- +goose Down
DROP TABLE admin_schedule;
