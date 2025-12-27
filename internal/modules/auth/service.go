package auth

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shubhbham/uptime-monitor/internal/auth"
	"github.com/shubhbham/uptime-monitor/internal/config"
	"github.com/shubhbham/uptime-monitor/internal/domain"
)

type Service struct {
	repo          *Repository
	clerkVerifier *auth.ClerkVerifier
	clerkClient   *auth.ClerkClient
	config        *config.AuthConfig
}

func NewService(repo *Repository, cfg *config.AuthConfig) *Service {
	return &Service{
		repo:          repo,
		clerkVerifier: auth.NewClerkVerifier(cfg.ClerkJWKSURL),
		clerkClient:   auth.NewClerkClient(cfg.ClerkSecretKey),
		config:        cfg,
	}
}

// VerifyClerkToken verifies a Clerk JWT and returns/creates user
func (s *Service) VerifyClerkToken(ctx context.Context, token string) (*domain.AuthContext, error) {
	// Verify JWT token
	claims, err := s.clerkVerifier.VerifyToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	clerkUserID := claims.GetUserID()

	// Check if user exists
	user, err := s.repo.GetUserByClerkID(ctx, clerkUserID)
	if err != nil {
		if err == pgx.ErrNoRows {
			// User doesn't exist - fetch from Clerk and create
			user, err = s.createUserFromClerk(ctx, clerkUserID)
			if err != nil {
				return nil, fmt.Errorf("failed to create user: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is deactivated")
	}

	// If email or name is empty, fetch from Clerk and update
	if user.Email == "" || user.Name == "" {
		if err := s.enrichUserDataFromClerk(ctx, user); err != nil {
			// Log error but don't fail authentication
			log.Printf("Failed to enrich user data from Clerk: %v", err)
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

// createUserFromClerk fetches user data from Clerk API and creates user
func (s *Service) createUserFromClerk(ctx context.Context, clerkUserID string) (*domain.User, error) {
	// Fetch user details from Clerk
	clerkUser, err := s.clerkClient.GetUser(ctx, clerkUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user from Clerk: %w", err)
	}

	// Create user record
	user := &domain.User{
		UserID:      clerkUserID,
		UserType:    "clerk",
		Email:       clerkUser.GetEmail(),
		Name:        clerkUser.GetFullName(),
		ClerkUserID: &clerkUserID,
		IsActive:    true,
	}

	// If name is still empty, use email as name
	if user.Name == "" && user.Email != "" {
		user.Name = user.Email
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user in database: %w", err)
	}

	log.Printf("Created new user from Clerk: %s (%s)", user.UserID, user.Email)

	return user, nil
}

// enrichUserDataFromClerk updates user email and name from Clerk
func (s *Service) enrichUserDataFromClerk(ctx context.Context, user *domain.User) error {
	if user.ClerkUserID == nil {
		return nil // Not a Clerk user
	}

	// Fetch user details from Clerk
	clerkUser, err := s.clerkClient.GetUser(ctx, *user.ClerkUserID)
	if err != nil {
		return err
	}

	// Update user data
	email := clerkUser.GetEmail()
	name := clerkUser.GetFullName()

	// If name is empty, use email
	if name == "" && email != "" {
		name = email
	}

	// Update only if we got new data
	if email != "" {
		user.Email = email
	}
	if name != "" {
		user.Name = name
	}

	// Update in database
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return err
	}

	log.Printf("Enriched user data from Clerk: %s", user.UserID)

	return nil
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

	// Additional check: verify user is still active
	isActive, err := s.repo.IsUserActive(ctx, apiKey.UserID)
	if err != nil {
		return nil, fmt.Errorf("user verification failed")
	}
	if !isActive {
		return nil, fmt.Errorf("user account is deactivated")
	}

	// Update usage tracking (async, best-effort)
	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
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
	// Verify user is active
	isActive, err := s.repo.IsUserActive(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	if !isActive {
		return nil, fmt.Errorf("cannot create API key for inactive user")
	}

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
		req.RateLimitPerHour = 3000
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

// DeleteUser deletes a user and all associated data (CASCADE)
func (s *Service) DeleteUser(ctx context.Context, userID string) error {
	return s.repo.DeleteUser(ctx, userID)
}

// DeactivateUser deactivates a user (soft delete)
func (s *Service) DeactivateUser(ctx context.Context, userID string) error {
	return s.repo.DeactivateUser(ctx, userID)
}

// GetUserStats returns statistics for the authenticated user
func (s *Service) GetUserStats(ctx context.Context, userID string) (*domain.UserStats, error) {
	return s.repo.GetUserStats(ctx, userID)
}
