package auth_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/subhayan-bhattacharya/events/api/internal/auth"
)

// fakeRepository is an in-memory auth.Repository for service tests. It lets
// the tests run without a database and has two jobs:
//   - script: the fields in the first group decide what the fake returns;
//   - spy: the fields in the second group record what the service passed in,
//     so tests can assert on it afterward.
//
// It is not safe for concurrent use; the tests call it sequentially.
type fakeRepository struct {
	// userID is the ID UpsertUser returns on success. Tests should set a
	// non-zero value so "the same ID reaches InsertSession" is a real check.
	userID uuid.UUID
	// upsertErr, if set, makes UpsertUser fail with this error.
	upsertErr error
	// insertErr, if set, makes InsertSession fail with this error.
	insertErr error

	// calls lists the methods called, in order ("upsert user",
	// "insert session"), to check that upsert happens before insert and that
	// insert is skipped when upsert fails.
	calls []string
	// upsertContexts and insertContexts hold the context each call received.
	upsertContexts []context.Context
	insertContexts []context.Context
	// upsertUsers holds every identity passed to UpsertUser.
	upsertUsers []auth.UserIdentity
	// insertedSessions holds every session passed to InsertSession (hash,
	// user ID, expiry; never the raw token).
	insertedSessions []auth.TokenDetails
}

// Compile-time check that fakeRepository satisfies auth.Repository. If the
// interface gains a method, this line fails to compile here.
var _ auth.Repository = (*fakeRepository)(nil)

// UpsertUser records the call and returns the scripted error or userID.
// On error it returns uuid.Nil, as the real repository does.
func (r *fakeRepository) UpsertUser(
	ctx context.Context,
	user auth.UserIdentity,
) (uuid.UUID, error) {
	r.calls = append(r.calls, "upsert user")
	r.upsertContexts = append(r.upsertContexts, ctx)
	r.upsertUsers = append(r.upsertUsers, user)
	if r.upsertErr != nil {
		return uuid.Nil, r.upsertErr
	}
	return r.userID, nil
}

// InsertSession records the call and returns the scripted error (nil means
// success).
func (r *fakeRepository) InsertSession(
	ctx context.Context,
	session auth.TokenDetails,
) error {
	r.calls = append(r.calls, "insert session")
	r.insertContexts = append(r.insertContexts, ctx)
	r.insertedSessions = append(r.insertedSessions, session)
	return r.insertErr
}

func TestCreateSessionReturnsRawTokenAndStoresHash(t *testing.T) {
	userID := uuid.MustParse("59528d0e-70e6-4bc4-8d05-13264a8a2980")
	repository := &fakeRepository{}
	service := auth.NewService(repository)
	ctx := context.Background()

	before := time.Now()
	tokenData, err := service.CreateSession(ctx, userID)
	after := time.Now()
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	wantCalls := []string{"insert session"}
	if !slices.Equal(repository.calls, wantCalls) {
		t.Fatalf(
			"repository calls: got %v, want %v", repository.calls, wantCalls,
		)
	}
	if got := len(repository.insertedSessions); got != 1 {
		t.Fatalf("insert calls: got %d, want 1", got)
	}

	rawToken, err := base64.URLEncoding.DecodeString(tokenData.Token)
	if err != nil {
		t.Fatalf("decode raw token: %v", err)
	}
	if got, want := len(rawToken), 32; got != want {
		t.Errorf("raw token length: got %d bytes, want %d", got, want)
	}

	insertedSession := repository.insertedSessions[0]
	if insertedSession.UserID != userID {
		t.Errorf(
			"inserted session user ID: got %s, want %s",
			insertedSession.UserID,
			userID,
		)
	}
	wantHash := sha256.Sum256([]byte(tokenData.Token))
	if !bytes.Equal(insertedSession.TokenHash, wantHash[:]) {
		t.Errorf(
			"inserted token hash: got %x, want %x",
			insertedSession.TokenHash,
			wantHash,
		)
	}
	if bytes.Equal(insertedSession.TokenHash, []byte(tokenData.Token)) {
		t.Error("inserted token hash contains the raw token")
	}

	if !insertedSession.ExpiresAt.Equal(tokenData.Expiration) {
		t.Errorf(
			"inserted expiry: got %s, want %s",
			insertedSession.ExpiresAt,
			tokenData.Expiration,
		)
	}
	earliestExpiry := before.Add(7 * 24 * time.Hour)
	latestExpiry := after.Add(7 * 24 * time.Hour)
	if tokenData.Expiration.Before(earliestExpiry) ||
		tokenData.Expiration.After(latestExpiry) {
		t.Errorf(
			"expiration: got %s, want between %s and %s",
			tokenData.Expiration,
			earliestExpiry,
			latestExpiry,
		)
	}

	if got := len(repository.insertContexts); got != 1 {
		t.Fatalf("insert contexts: got %d, want 1", got)
	}
	if repository.insertContexts[0] != ctx {
		t.Error("insert session did not receive the caller context")
	}
}

func TestCreateSessionCreatesDistinctTokens(t *testing.T) {
	userID := uuid.MustParse("68d79072-980a-4550-bfc1-e49329a49475")
	repository := &fakeRepository{}
	service := auth.NewService(repository)

	firstToken, err := service.CreateSession(context.Background(), userID)
	if err != nil {
		t.Fatalf("create first session: %v", err)
	}
	secondToken, err := service.CreateSession(context.Background(), userID)
	if err != nil {
		t.Fatalf("create second session: %v", err)
	}

	if firstToken.Token == secondToken.Token {
		t.Error("successive sessions returned the same raw token")
	}
	if got := len(repository.insertedSessions); got != 2 {
		t.Fatalf("inserted sessions: got %d, want 2", got)
	}
	if bytes.Equal(
		repository.insertedSessions[0].TokenHash,
		repository.insertedSessions[1].TokenHash,
	) {
		t.Error("successive sessions stored the same token hash")
	}
}

func TestDevLoginCreatesSessionForConfiguredDevUser(t *testing.T) {
	userID := uuid.MustParse("7c0efb4d-8298-46cd-8e4d-e6eb25327d46")
	repository := &fakeRepository{userID: userID}
	service := auth.NewService(repository)
	config := auth.DevUserConfig{
		Email:       "developer@example.com",
		DisplayName: "Local Developer",
		TimeZone:    "Europe/Berlin",
	}

	_, err := service.DevLogin(context.Background(), config)
	if err != nil {
		t.Fatalf("dev login: %v", err)
	}

	wantCalls := []string{"upsert user", "insert session"}
	if !slices.Equal(repository.calls, wantCalls) {
		t.Fatalf(
			"repository calls: got %v, want %v", repository.calls, wantCalls,
		)
	}

	if got := len(repository.upsertUsers); got != 1 {
		t.Fatalf("upsert calls: got %d, want 1", got)
	}
	wantIdentity := auth.UserIdentity{
		Issuer:      "dev",
		Subject:     "local",
		Email:       config.Email,
		DisplayName: config.DisplayName,
		TimeZone:    config.TimeZone,
	}
	if got := repository.upsertUsers[0]; got != wantIdentity {
		t.Errorf("upsert identity: got %+v, want %+v", got, wantIdentity)
	}

	if got := len(repository.insertedSessions); got != 1 {
		t.Fatalf("insert calls: got %d, want 1", got)
	}
	if got := repository.insertedSessions[0].UserID; got != userID {
		t.Errorf("inserted session user ID: got %s, want %s", got, userID)
	}
}

func TestDevLoginStopsWhenUpsertFails(t *testing.T) {
	upsertErr := errors.New("upsert user")
	repository := &fakeRepository{upsertErr: upsertErr}
	service := auth.NewService(repository)
	config := auth.DevUserConfig{
		Email:       "developer@example.com",
		DisplayName: "Local Developer",
		TimeZone:    "Europe/Berlin",
	}

	tokenData, err := service.DevLogin(context.Background(), config)
	if !errors.Is(err, upsertErr) {
		t.Fatalf("dev login error: got %v, want %v", err, upsertErr)
	}
	if tokenData != (auth.TokenData{}) {
		t.Errorf("token data: got %+v, want zero value", tokenData)
	}

	wantCalls := []string{"upsert user"}
	if !slices.Equal(repository.calls, wantCalls) {
		t.Errorf(
			"repository calls: got %v, want %v", repository.calls, wantCalls,
		)
	}
	if got := len(repository.insertedSessions); got != 0 {
		t.Errorf("inserted sessions: got %d, want 0", got)
	}
}

func TestDevLoginReturnsNoTokenWhenSessionInsertFails(t *testing.T) {
	userID := uuid.MustParse("a3efaf4a-741a-45a7-ae5e-0b9bda85896b")
	insertErr := errors.New("insert session")
	repository := &fakeRepository{
		userID:    userID,
		insertErr: insertErr,
	}
	service := auth.NewService(repository)
	config := auth.DevUserConfig{
		Email:       "developer@example.com",
		DisplayName: "Local Developer",
		TimeZone:    "Europe/Berlin",
	}

	tokenData, err := service.DevLogin(context.Background(), config)
	if !errors.Is(err, insertErr) {
		t.Fatalf("dev login error: got %v, want wrapped %v", err, insertErr)
	}
	if tokenData != (auth.TokenData{}) {
		t.Errorf("token data: got %+v, want zero value", tokenData)
	}

	wantCalls := []string{"upsert user", "insert session"}
	if !slices.Equal(repository.calls, wantCalls) {
		t.Errorf(
			"repository calls: got %v, want %v", repository.calls, wantCalls,
		)
	}
	if got := len(repository.insertedSessions); got != 1 {
		t.Errorf("insert attempts: got %d, want 1", got)
	}
}
