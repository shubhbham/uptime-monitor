package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/shubhbham/uptime-monitor/internal/auth"
	"github.com/shubhbham/uptime-monitor/internal/config"
	"github.com/shubhbham/uptime-monitor/internal/domain"
)

type Service struct {
	repo          *Repository
	clerkVerifier *auth.ClerkVerifier
	config        *config.AuthConfig
}

func NewService(repo *Repository, cfg *config.AuthConfig) *Service {
	return &Service{
		repo:          repo,
		clerkVerifier: auth.NewClerkVerifier(cfg.ClerkJWKSURL),
		config:        cfg,
	}
}

// VerifyClerkToken verifies a Clerk JWT and returns/creates user
func (s *Service) VerifyClerkToken(ctx context.Context, token string) (*domain.AuthContext, error) {
	claims, err := s.clerkVerifier.VerifyToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Check if user exists, if not create
	user, err := s.repo.GetUserByClerkID(ctx, claims.GetUserID())
	if err != nil {
		// User doesn't exist, create new user
		user = &domain.User{
			UserID:      claims.GetUserID(),
			UserType:    "clerk",
			Email:       claims.GetEmail(),
			Name:        claims.GetName(),
			ClerkUserID: &claims.Subject,
			IsActive:    true,
		}

		if err := s.repo.CreateUser(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	return &domain.AuthContext{
		UserID:   user.UserID,
		UserType: "clerk",
		Email:    user.Email,
		Name:     user.Name,
		Scopes:   []string{"*"}, // Clerk users have all scopes
	}, nil
}

// VerifyAPIKey verifies an API key and returns auth context
func (s *Service) VerifyAPIKey(ctx context.Context, rawKey string) (*domain.AuthContext, error) {
	if err := auth.ValidateAPIKeyFormat(rawKey); err != nil {
		return nil, err
	}

	keyHash := auth.HashAPIKey(rawKey)
	apiKey, err := s.repo.GetAPIKeyByHash(ctx, keyHash)
	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	// Update usage tracking (async, don't block on errors)
	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.repo.UpdateAPIKeyUsage(updateCtx, apiKey.ID)
	}()

	return &domain.AuthContext{
		UserID:   apiKey.UserID,
		UserType: "api_customer",
		Scopes:   apiKey.Scopes,
	}, nil
}

// CreateAPIKey creates a new API key for a user
func (s *Service) CreateAPIKey(ctx context.Context, userID string, req *domain.CreateAPIKeyRequest) (*domain.CreateAPIKeyResponse, error) {
	// Generate raw API key
	rawKey, err := auth.GenerateAPIKey(s.config.APIKeyPrefix, s.config.APIKeyLength)
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Hash the key for storage
	keyHash := auth.HashAPIKey(rawKey)
	keyPrefix := auth.ExtractKeyPrefix(rawKey)

	// Set defaults
	if req.RateLimitPerHour == 0 {
		req.RateLimitPerHour = 1000
	}
	if len(req.Scopes) == 0 {
		req.Scopes = []string{"monitors:read", "monitors:write", "incidents:read", "stats:read"}
	}

	apiKey := &domain.APIKey{
		UserID:           userID,
		KeyHash:          keyHash,
		Name:             req.Name,
		KeyPrefix:        keyPrefix,
		Scopes:           req.Scopes,
		RateLimitPerHour: req.RateLimitPerHour,
		IsActive:         true,
		ExpiresAt:        req.ExpiresAt,
	}

	if err := s.repo.CreateAPIKey(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	return &domain.CreateAPIKeyResponse{
		APIKey: *apiKey,
		RawKey: rawKey, // Only returned once
	}, nil
}

// ListAPIKeys lists all API keys for a user
func (s *Service) ListAPIKeys(ctx context.Context, userID string) ([]*domain.APIKey, error) {
	return s.repo.ListAPIKeysByUser(ctx, userID)
}

// DeleteAPIKey deletes an API key
func (s *Service) DeleteAPIKey(ctx context.Context, keyID, userID string) error {
	return s.repo.DeleteAPIKey(ctx, keyID, userID)
}

// UpdateAPIKeyStatus updates the active status of an API key
func (s *Service) UpdateAPIKeyStatus(ctx context.Context, keyID, userID string, isActive bool) error {
	return s.repo.UpdateAPIKey(ctx, keyID, userID, isActive)
}

// GetOrCreateUser gets or creates a user
func (s *Service) GetOrCreateUser(ctx context.Context, userID, userType, email, name string) (*domain.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		user = &domain.User{
			UserID:   userID,
			UserType: userType,
			Email:    email,
			Name:     name,
			IsActive: true,
		}
		if err := s.repo.CreateUser(ctx, user); err != nil {
			return nil, err
		}
	}
	return user, nil
}