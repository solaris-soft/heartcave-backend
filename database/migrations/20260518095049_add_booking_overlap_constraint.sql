-- +goose Up
CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE bookings
  ADD CONSTRAINT no_overlapping_active_bookings
  EXCLUDE USING gist (
    tsrange(starts_at, ends_at, '[)') WITH &&
  ) WHERE (status != 'cancelled');

-- +goose Down
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS no_overlapping_active_bookings;
