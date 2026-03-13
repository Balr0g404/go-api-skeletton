CREATE TABLE IF NOT EXISTS users (
    id          BIGSERIAL PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    email       VARCHAR(255) NOT NULL,
    password    VARCHAR(255) NOT NULL,
    first_name  VARCHAR(100) NOT NULL DEFAULT '',
    last_name   VARCHAR(100) NOT NULL DEFAULT '',
    role        VARCHAR(20)  NOT NULL DEFAULT 'user',
    active      BOOLEAN      NOT NULL DEFAULT TRUE
);

-- Partial unique index: allows re-registering a soft-deleted email address.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email
    ON users (email)
    WHERE deleted_at IS NULL;

-- Index used by GORM soft-delete queries.
CREATE INDEX IF NOT EXISTS idx_users_deleted_at
    ON users (deleted_at);
