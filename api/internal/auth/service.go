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

type DevUserConfig struct {
	Email       string
	DisplayName string
	TimeZone    string
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

func GetTokenDetails(userId uuid.UUID) (TokenDetailsHandler, error) {
	var tokenDetails TokenDetailsHandler
	token, err := GenerateSessionToken()
	if err != nil {
		return tokenDetails, err
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	tokenDetails.Token = token
	tokenDetails.Expiration = expiresAt
	return tokenDetails, nil
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s Service) DevLogin(
	ctx context.Context,
	devUserDetails DevUserConfig,
) (TokenDetailsHandler, error) {
	userDetailsIdentity := UserIdentity{
		Issuer:      "dev",
		Subject:     "local",
		Email:       devUserDetails.Email,
		DisplayName: devUserDetails.DisplayName,
		TimeZone:    devUserDetails.TimeZone,
	}
	userId, err := s.repository.UpsertUser(ctx, userDetailsIdentity)
	if err != nil {
		return TokenDetailsHandler{}, err
	}
	return s.CreateSession(ctx, userId)
}

func (s Service) CreateSession(
	ctx context.Context, userID uuid.UUID,
) (TokenDetailsHandler, error) {
	var tokenDetails TokenDetailsHandler
	tokenDetails, err := GetTokenDetails(userID)
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
		return TokenDetailsHandler{}, fmt.Errorf("save session: %w", err)
	}
	return tokenDetails, nil
}
