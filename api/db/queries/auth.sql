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

-- name: AuthenticateSession :one
-- Successful authentication refreshes last_seen_at. The fixed expires_at
-- remains unchanged, so activity cannot extend the seven-day maximum.
-- No row (pgx.ErrNoRows) means unknown, absolutely expired, or idle for
-- at least 24 hours: the caller answers 401. Go supplies @now for testing.
WITH refreshed AS (
    UPDATE sessions
    SET last_seen_at = GREATEST(last_seen_at, @now::timestamptz)
    WHERE token_hash = @token_hash
      AND expires_at > @now::timestamptz
      AND last_seen_at + interval '24 hours' > @now::timestamptz
    RETURNING user_id, expires_at
)
SELECT u.id, u.email, u.display_name, u.time_zone, u.preferred_currency,
       refreshed.expires_at
FROM refreshed
JOIN users AS u ON u.id = refreshed.user_id;

-- name: DeleteSession :exec
-- Logout. Deleting an already-deleted session is not an error.
DELETE FROM sessions
WHERE token_hash = @token_hash;
