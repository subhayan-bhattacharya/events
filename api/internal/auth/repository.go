package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/subhayan-bhattacharya/events/api/internal/db"
)

// TokenDetails is what is stored for a session: the hash of its token, the
// owning user and the absolute expiry.
type TokenDetails struct {
	TokenHash []byte
	UserID    uuid.UUID
	ExpiresAt time.Time
}

// UserIdentity identifies an account by OIDC issuer and subject (D4) and
// carries the profile fields written with it.
type UserIdentity struct {
	Issuer      string
	Subject     string
	Email       string
	DisplayName string
	TimeZone    string
}

// SessionRepository implements Repository on top of the sqlc queries.
type SessionRepository struct {
	queries *db.Queries
}

// NewSessionRepository returns a SessionRepository using queries.
func NewSessionRepository(queries *db.Queries) *SessionRepository {
	return &SessionRepository{queries: queries}
}

// InsertSession stores a session row for the given token hash.
func (r *SessionRepository) InsertSession(
	ctx context.Context, tokenDetails TokenDetails,
) error {
	return r.queries.CreateSession(
		ctx, db.CreateSessionParams{
			TokenHash: tokenDetails.TokenHash,
			UserID:    tokenDetails.UserID,
			ExpiresAt: pgtype.Timestamptz{
				Time: tokenDetails.ExpiresAt, Valid: true,
			},
		},
	)
}

// UpsertUser inserts the user for the identity, or updates email, display name
// and time zone if the identity exists, and returns the user's ID. The display
// name is always stored as non-NULL, even when empty.
func (r *SessionRepository) UpsertUser(
	ctx context.Context, userDetails UserIdentity,
) (uuid.UUID, error) {
	user, err := r.queries.UpsertUserByIdentity(
		ctx, db.UpsertUserByIdentityParams{
			OidcIssuer:  userDetails.Issuer,
			OidcSubject: userDetails.Subject,
			Email:       userDetails.Email,
			DisplayName: pgtype.Text{
				String: userDetails.DisplayName, Valid: true,
			},
			TimeZone: userDetails.TimeZone,
		},
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert user: %w", err)
	}
	return user.ID, nil
}
