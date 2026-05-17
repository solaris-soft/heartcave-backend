-- +goose Up
CREATE TABLE refresh_tokens (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
token_hash VARCHAR(255) NOT NULL UNIQUE,
expires_at TIMESTAMP NOT NULL,
created_at TIMESTAMP NOT NULL DEFAULT now(),
revoked_at TIMESTAMP,
ip_address VARCHAR(45),
user_agent TEXT
);

CREATE INDEX idx_refresh_tokens_user_id
ON refresh_tokens (user_id);

CREATE INDEX idx_refresh_token_hash
ON refresh_tokens (token_hash);

-- +goose Down
DROP TABLE refresh_tokens;
