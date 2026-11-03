package auth

import (
	"context"
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

type SessionRepository struct {
	queries *db.Queries
}

func NewSessionRepository(queries *db.Queries) *SessionRepository {
	return &SessionRepository{queries: queries}
}

func (r *SessionRepository) InsertSession(ctx context.Context, tokenDetails TokenDetails) error {
	return r.queries.CreateSession(ctx, db.CreateSessionParams{
		TokenHash: tokenDetails.TokenHash,
		UserID:    tokenDetails.UserID,
		ExpiresAt: pgtype.Timestamptz{Time: tokenDetails.ExpiresAt, Valid: true},
	})
}
