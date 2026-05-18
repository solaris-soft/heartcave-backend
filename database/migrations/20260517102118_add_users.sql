-- +goose Up
CREATE TYPE user_role AS ENUM ('customer', 'admin');

CREATE TABLE users (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
name TEXT NOT NULL,
email TEXT NOT NULL UNIQUE,
role user_role NOT NULL DEFAULT 'customer',
password_hash TEXT NOT NULL,
created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE users;
DROP TYPE user_role;
