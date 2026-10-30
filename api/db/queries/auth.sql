-- name: UpsertUserByIdentity :one
-- Ticket 05; D4: identity is issuer + subject. Creates the user for this
-- identity, or refreshes the profile fields if the identity already exists. Used by dev login (ticket 03) with the fixed
-- configured identity. A different identity with the same email still fails
-- on uq_users_email.
INSERT INTO users (oidc_issuer, oidc_subject, email, display_name, time_zone)
VALUES (@oidc_issuer, @oidc_subject, @email, @display_name, @time_zone)
ON CONFLICT ON CONSTRAINT uq_users_oidc_identity DO UPDATE
SET email        = EXCLUDED.email,
    display_name = EXCLUDED.display_name,
    time_zone    = EXCLUDED.time_zone
RETURNING id, oidc_issuer, oidc_subject, email, display_name, time_zone,
          preferred_currency, created_at, updated_at;

-- name: CreateSession :exec
-- token_hash is the SHA-256 of the random cookie token; expires_at is set by Go.
INSERT INTO sessions (token_hash, user_id, expires_at)
VALUES (@token_hash, @user_id, @expires_at);

-- name: GetSessionUser :one
-- Finds a session that has not expired at @now, with its user. No row
-- (pgx.ErrNoRows) means unknown or expired: the caller answers 401.
-- @now comes from Go so tests can pass a fixed time.
SELECT u.id, u.email, u.display_name, u.time_zone, u.preferred_currency,
       s.expires_at
FROM sessions AS s
JOIN users AS u ON u.id = s.user_id
WHERE s.token_hash = @token_hash
  AND s.expires_at > @now::timestamptz;

-- name: DeleteSession :exec
-- Logout. Deleting an already-deleted session is not an error.
DELETE FROM sessions
WHERE token_hash = @token_hash;
