-- +goose Up
CREATE TABLE service_availability (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
service_id UUID NOT NULL REFERENCES services(id),
day_of_week SMALLINT NOT NULL CHECK(
    day_of_week BETWEEN 0 AND 6
    ),
start_time TIME NOT NULL,
end_time TIME NOT NULL,
created_at TIMESTAMP NOT NULL DEFAULT now(),
updated_at TIMESTAMP NOT NULL DEFAULT now(),

CHECK (start_time < end_time)
);

-- +goose Down
DROP TABLE service_availability;
