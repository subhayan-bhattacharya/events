package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository is the persistence the Service needs. It is defined here, by the
// consumer, so tests can supply a fake.
type Repository interface {
	// InsertSession stores a session row keyed by the hash of its token.
	InsertSession(ctx context.Context, tokenDetails TokenDetails) error
	// UpsertUser creates the user for an identity or refreshes its profile,
	// and returns the user's ID.
	UpsertUser(ctx context.Context, userDetails UserIdentity) (uuid.UUID, error)
}

// TokenData is a newly issued session: the raw token that goes into
// the cookie and the time the session expires. The raw token is never stored.
type TokenData struct {
	Token      string
	Expiration time.Time
}

// DevUserConfig is the fixed profile used by dev login. It comes from server
// configuration, never from an HTTP request.
type DevUserConfig struct {
	Email       string
	DisplayName string
	TimeZone    string
}

// newSessionToken returns 32 random bytes (256 bits) from crypto/rand,
// URL-base64 encoded.
func newSessionToken() (string, error) {
	b := make([]byte, 32) // 256 bits of entropy
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	// Base64 URLEncoding ensures the token is safe to use in HTTP headers/cookies
	return base64.URLEncoding.EncodeToString(b), nil
}

// newTokenDetails generates a session token and sets its expiry to seven days
// from now (the absolute session lifetime).
func newTokenDetails() (TokenData, error) {
	var tokenDetails TokenData
	token, err := newSessionToken()
	if err != nil {
		return TokenData{}, fmt.Errorf(
			"session token generation error: %w",
			err,
		)
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	tokenDetails.Token = token
	tokenDetails.Expiration = expiresAt
	return tokenDetails, nil
}

// Service holds the login and session rules of the auth package.
type Service struct {
	repository Repository
}

// NewService returns a Service that persists through repository.
func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

// DevLogin signs in the fixed development identity (issuer "dev", subject
// "local"). It upserts the user with the configured profile, then creates a
// new session for it. If the upsert fails, no session is created. The two
// steps are not in one transaction, so a user may remain if session insertion
// fails. Repeated calls reuse the same user and add another session.
func (s *Service) DevLogin(
	ctx context.Context,
	devUserDetails DevUserConfig,
) (TokenData, error) {
	const issuer = "dev"
	const subject = "local"
	userDetailsIdentity := UserIdentity{
		Issuer:      issuer,
		Subject:     subject,
		Email:       devUserDetails.Email,
		DisplayName: devUserDetails.DisplayName,
		TimeZone:    devUserDetails.TimeZone,
	}
	userID, err := s.repository.UpsertUser(ctx, userDetailsIdentity)
	if err != nil {
		return TokenData{}, err
	}
	return s.CreateSession(ctx, userID)
}

// CreateSession issues a new session for userID. Only the SHA-256 hash of the
// token is stored; the raw token and expiry are returned for the cookie.
func (s *Service) CreateSession(
	ctx context.Context, userID uuid.UUID,
) (TokenData, error) {
	tokenDetails, err := newTokenDetails()
	if err != nil {
		return tokenDetails, err
	}
	tokenHash := sha256.Sum256([]byte(tokenDetails.Token))
	repositoryTokenDetails := TokenDetails{
		TokenHash: tokenHash[:],
		UserID:    userID,
		ExpiresAt: tokenDetails.Expiration,
	}
	err = s.repository.InsertSession(ctx, repositoryTokenDetails)
	if err != nil {
		return TokenData{}, fmt.Errorf("save session: %w", err)
	}
	return tokenDetails, nil
}
