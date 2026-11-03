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

type TokenDetailsHandler struct {
	Token      string
	Expiration time.Time
}

// GenerateSessionToken creates a 32-byte cryptographically secure random string
func GenerateSessionToken() (string, error) {
	b := make([]byte, 32) // 256 bits of entropy
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	// Base64 URLEncoding ensures the token is safe to use in HTTP headers/cookies
	return base64.URLEncoding.EncodeToString(b), nil
}

type Repository interface {
	InsertSession(ctx context.Context, tokenDetails TokenDetails) error
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s Service) CreateSession(ctx context.Context, userID uuid.UUID) (TokenDetailsHandler, error) {
	var tokenDetails TokenDetailsHandler
	token, err := GenerateSessionToken()
	if err != nil {
		return tokenDetails, fmt.Errorf("generate session token: %w", err)
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	tokenDetails.Token = token
	tokenDetails.Expiration = expiresAt
	tokenHash := sha256.Sum256([]byte(token))
	repositoryTokenDetails := TokenDetails{
		TokenHash: tokenHash[:],
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
	err = s.repository.InsertSession(ctx, repositoryTokenDetails)
	if err != nil {
		return TokenDetailsHandler{}, fmt.Errorf("save session: %w", err)
	}
	return tokenDetails, nil
}
