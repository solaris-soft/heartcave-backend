-- +goose Up
CREATE TABLE services (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
name TEXT NOT NULL UNIQUE,
price DECIMAL(19, 2) NOT NULL,
description TEXT NOT NULL,
session_minutes INTEGER NOT NULL,
created_at TIMESTAMP NOT NULL DEFAULT now(),
updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE services;
