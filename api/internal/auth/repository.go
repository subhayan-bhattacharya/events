package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/subhayan-bhattacharya/events/api/internal/db"
)

type TokenDetails struct {
	TokenHash []byte
	UserID    uuid.UUID
	ExpiresAt time.Time
}

type UserIdentity struct {
	Issuer      string
	Subject     string
	Email       string
	DisplayName string
	TimeZone    string
}

type SessionRepository struct {
	queries *db.Queries
}

func NewSessionRepository(queries *db.Queries) *SessionRepository {
	return &SessionRepository{queries: queries}
}

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
