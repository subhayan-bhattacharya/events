package auth_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/subhayan-bhattacharya/events/api/internal/auth"
	"github.com/subhayan-bhattacharya/events/api/internal/db"
	"github.com/subhayan-bhattacharya/events/api/internal/testdb"
)

func TestSessionRepositoryPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	database, err := testdb.StartMigrated(ctx)
	if err != nil {
		t.Fatalf("start migrated test database: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := database.Close(cleanupCtx); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	t.Run("upsert inserts and refreshes one identity", func(t *testing.T) {
		tx, err := database.Begin(ctx)
		if err != nil {
			t.Fatalf("begin repository test transaction: %v", err)
		}
		t.Cleanup(func() {
			rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer rollbackCancel()
			if err := tx.Rollback(rollbackCtx); err != nil {
				t.Errorf("roll back repository test transaction: %v", err)
			}
		})

		repository := auth.NewSessionRepository(db.New(tx))
		assertStoredUser := func(userID uuid.UUID, want auth.UserIdentity) {
			t.Helper()
			var got auth.UserIdentity
			if err := tx.QueryRow(
				ctx,
				`SELECT oidc_issuer, oidc_subject, email, display_name, time_zone
				 FROM users
				 WHERE id = $1`,
				userID,
			).Scan(
				&got.Issuer,
				&got.Subject,
				&got.Email,
				&got.DisplayName,
				&got.TimeZone,
			); err != nil {
				t.Fatalf("read stored user: %v", err)
			}
			if got != want {
				t.Errorf("stored user: got %+v, want %+v", got, want)
			}
		}

		original := auth.UserIdentity{
			Issuer:      "dev",
			Subject:     "local",
			Email:       "original-developer@example.com",
			DisplayName: "Original Developer",
			TimeZone:    "UTC",
		}
		userID, err := repository.UpsertUser(ctx, original)
		if err != nil {
			t.Fatalf("insert user: %v", err)
		}
		if userID == uuid.Nil {
			t.Fatal("insert user returned a nil UUID")
		}
		assertStoredUser(userID, original)

		updated := auth.UserIdentity{
			Issuer:      original.Issuer,
			Subject:     original.Subject,
			Email:       "developer@example.com",
			DisplayName: "Local Developer",
			TimeZone:    "Europe/Berlin",
		}
		updatedUserID, err := repository.UpsertUser(ctx, updated)
		if err != nil {
			t.Fatalf("update user: %v", err)
		}
		if updatedUserID != userID {
			t.Errorf("updated user ID: got %s, want %s", updatedUserID, userID)
		}
		assertStoredUser(userID, updated)

		var identityCount int
		if err := tx.QueryRow(
			ctx,
			`SELECT count(*)
			 FROM users
			 WHERE oidc_issuer = $1 AND oidc_subject = $2`,
			updated.Issuer,
			updated.Subject,
		).Scan(&identityCount); err != nil {
			t.Fatalf("count stored identity rows: %v", err)
		}
		if identityCount != 1 {
			t.Errorf("stored identity rows: got %d, want 1", identityCount)
		}
	})

	t.Run("upsert rejects a case-insensitive email collision", func(t *testing.T) {
		tx, err := database.Begin(ctx)
		if err != nil {
			t.Fatalf("begin repository test transaction: %v", err)
		}
		t.Cleanup(func() {
			rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer rollbackCancel()
			if err := tx.Rollback(rollbackCtx); err != nil {
				t.Errorf("roll back repository test transaction: %v", err)
			}
		})

		repository := auth.NewSessionRepository(db.New(tx))
		firstIdentity := auth.UserIdentity{
			Issuer:      "dev",
			Subject:     "local",
			Email:       "Developer@Example.com",
			DisplayName: "Local Developer",
			TimeZone:    "Europe/Berlin",
		}
		if _, err := repository.UpsertUser(ctx, firstIdentity); err != nil {
			t.Fatalf("insert first identity: %v", err)
		}

		secondIdentity := auth.UserIdentity{
			Issuer:      "https://accounts.google.com",
			Subject:     "google-subject",
			Email:       "developer@example.com",
			DisplayName: "Google Developer",
			TimeZone:    "Europe/Berlin",
		}
		userID, err := repository.UpsertUser(ctx, secondIdentity)
		if err == nil {
			t.Fatal("insert second identity with duplicate email succeeded; want an error")
		}
		if userID != uuid.Nil {
			t.Errorf("failed upsert user ID: got %s, want nil UUID", userID)
		}

		var postgresError *pgconn.PgError
		if !errors.As(err, &postgresError) {
			t.Fatalf("duplicate email error: got %T, want *pgconn.PgError", err)
		}
		if postgresError.Code != "23505" {
			t.Errorf("duplicate email SQLSTATE: got %s, want 23505", postgresError.Code)
		}
		if postgresError.ConstraintName != "uq_users_email" {
			t.Errorf(
				"duplicate email constraint: got %q, want %q",
				postgresError.ConstraintName,
				"uq_users_email",
			)
		}
	})

	t.Run("insert session persists its storage fields", func(t *testing.T) {
		tx, err := database.Begin(ctx)
		if err != nil {
			t.Fatalf("begin repository test transaction: %v", err)
		}
		t.Cleanup(func() {
			rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer rollbackCancel()
			if err := tx.Rollback(rollbackCtx); err != nil {
				t.Errorf("roll back repository test transaction: %v", err)
			}
		})

		repository := auth.NewSessionRepository(db.New(tx))
		userID, err := repository.UpsertUser(ctx, auth.UserIdentity{
			Issuer:      "dev",
			Subject:     "local",
			Email:       "developer@example.com",
			DisplayName: "Local Developer",
			TimeZone:    "Europe/Berlin",
		})
		if err != nil {
			t.Fatalf("insert session owner: %v", err)
		}

		tokenHash := bytes.Repeat([]byte{0x5a}, 32)
		expiresAt := time.Date(2030, time.May, 2, 12, 34, 56, 0, time.UTC)
		if err := repository.InsertSession(ctx, auth.TokenDetails{
			TokenHash: tokenHash,
			UserID:    userID,
			ExpiresAt: expiresAt,
		}); err != nil {
			t.Fatalf("insert session: %v", err)
		}

		var storedTokenHash []byte
		var storedUserID uuid.UUID
		var storedExpiresAt time.Time
		if err := tx.QueryRow(
			ctx,
			`SELECT token_hash, user_id, expires_at
			 FROM sessions
			 WHERE token_hash = $1`,
			tokenHash,
		).Scan(&storedTokenHash, &storedUserID, &storedExpiresAt); err != nil {
			t.Fatalf("read stored session: %v", err)
		}
		if !bytes.Equal(storedTokenHash, tokenHash) {
			t.Errorf("stored token hash: got %x, want %x", storedTokenHash, tokenHash)
		}
		if storedUserID != userID {
			t.Errorf("stored session user ID: got %s, want %s", storedUserID, userID)
		}
		if !storedExpiresAt.Equal(expiresAt) {
			t.Errorf("stored session expiry: got %s, want %s", storedExpiresAt, expiresAt)
		}
	})
}
