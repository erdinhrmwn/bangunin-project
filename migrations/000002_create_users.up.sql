-- id is UUIDv7, generated application-side (google/uuid NewV7) — Postgres 16
-- has no native uuidv7() generator, and app-side keeps ID generation
-- consistent across every table without a DB round-trip.
CREATE TABLE users (
    id                UUID PRIMARY KEY,
    role_id           SMALLINT NOT NULL REFERENCES roles(id),
    name              VARCHAR(150) NOT NULL,
    email             VARCHAR(255) NOT NULL UNIQUE,
    password_hash     VARCHAR(255) NOT NULL,
    phone             VARCHAR(20) NOT NULL DEFAULT '',
    status            VARCHAR(20) NOT NULL DEFAULT 'active',
    email_verified_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_role_id ON users(role_id);
