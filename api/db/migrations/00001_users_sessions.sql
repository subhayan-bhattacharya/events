-- Users and server-side sessions (ticket 05).
-- Decisions: D4 (OIDC identity), D10 (user time zone), D23 (hard delete),
-- D28 (server-side sessions). gen_random_uuid() is built into PostgreSQL 17,
-- so the pgcrypto extension is deliberately not installed.

-- +goose Up

-- Keeps updated_at current on every UPDATE. Later tables reuse this function.
-- +goose StatementBegin
CREATE FUNCTION set_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TABLE users (
    id                  UUID NOT NULL DEFAULT gen_random_uuid(),
    oidc_issuer         TEXT NOT NULL,
    oidc_subject        TEXT NOT NULL,
    email               TEXT NOT NULL,
    display_name        TEXT,
    time_zone           TEXT NOT NULL DEFAULT 'Europe/Berlin',
    preferred_currency  CHAR(3) NOT NULL DEFAULT 'EUR',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT pk_users PRIMARY KEY (id),
    -- D4: the identity key is issuer + subject, never email.
    CONSTRAINT uq_users_oidc_identity UNIQUE (oidc_issuer, oidc_subject)
);

-- D23: plain unique index (no soft delete), case-insensitive.
CREATE UNIQUE INDEX uq_users_email ON users (lower(email));

CREATE TRIGGER trg_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- D28: the cookie holds a random token; only its SHA-256 hash is stored here.
CREATE TABLE sessions (
    token_hash      BYTEA NOT NULL,
    user_id         UUID NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL,

    CONSTRAINT pk_sessions PRIMARY KEY (token_hash),
    -- D23: deleting a user deletes their sessions.
    CONSTRAINT fk_sessions_user FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_user ON sessions (user_id);
CREATE INDEX idx_sessions_expiry ON sessions (expires_at);

-- +goose Down
DROP TABLE sessions;
DROP TABLE users;
DROP FUNCTION set_updated_at();
