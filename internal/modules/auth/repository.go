package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shubhbham/uptime-monitor/internal/domain"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// User operations

func (r *Repository) CreateUser(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (user_id, user_type, email, name, clerk_user_id, company_name, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`

	return r.db.QueryRow(ctx, query,
		user.UserID, user.UserType, user.Email, user.Name,
		user.ClerkUserID, user.CompanyName, user.IsActive,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
}

func (r *Repository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	query := `
		SELECT user_id, user_type, email, name, clerk_user_id, company_name, is_active, created_at, updated_at
		FROM users
		WHERE user_id = $1
	`

	var user domain.User
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.UserID, &user.UserType, &user.Email, &user.Name,
		&user.ClerkUserID, &user.CompanyName, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) GetUserByClerkID(ctx context.Context, clerkUserID string) (*domain.User, error) {
	query := `
		SELECT user_id, user_type, email, name, clerk_user_id, company_name, is_active, created_at, updated_at
		FROM users
		WHERE clerk_user_id = $1
	`

	var user domain.User
	err := r.db.QueryRow(ctx, query, clerkUserID).Scan(
		&user.UserID, &user.UserType, &user.Email, &user.Name,
		&user.ClerkUserID, &user.CompanyName, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) DeleteUser(ctx context.Context, userID string) error {
	query := `DELETE FROM users WHERE user_id = $1`
	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *Repository) DeactivateUser(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET is_active = false, updated_at = NOW()
		WHERE user_id = $1
	`
	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// API Key operations

func (r *Repository) CreateAPIKey(ctx context.Context, apiKey *domain.APIKey) error {
	scopesJSON, err := json.Marshal(apiKey.Scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	query := `
		INSERT INTO api_keys (user_id, key_hash, name, key_prefix, scopes, rate_limit_per_hour, is_active, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRow(ctx, query,
		apiKey.UserID, apiKey.KeyHash, apiKey.Name, apiKey.KeyPrefix,
		scopesJSON, apiKey.RateLimitPerHour, apiKey.IsActive, apiKey.ExpiresAt,
	).Scan(&apiKey.ID, &apiKey.CreatedAt, &apiKey.UpdatedAt)
}

func (r *Repository) GetAPIKeyByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	query := `
		SELECT 
			ak.id, ak.user_id, ak.key_hash, ak.name, ak.key_prefix, ak.scopes, 
			ak.rate_limit_per_hour, ak.is_active, ak.expires_at, ak.last_used_at, 
			ak.total_requests, ak.created_at, ak.updated_at,
			u.is_active as user_is_active
		FROM api_keys ak
		INNER JOIN users u ON ak.user_id = u.user_id
		WHERE ak.key_hash = $1 
		  AND ak.is_active = true
		  AND u.is_active = true
	`

	var apiKey domain.APIKey
	var scopesJSON []byte
	var userIsActive bool

	err := r.db.QueryRow(ctx, query, keyHash).Scan(
		&apiKey.ID, &apiKey.UserID, &apiKey.KeyHash, &apiKey.Name, &apiKey.KeyPrefix,
		&scopesJSON, &apiKey.RateLimitPerHour, &apiKey.IsActive, &apiKey.ExpiresAt,
		&apiKey.LastUsedAt, &apiKey.TotalRequests, &apiKey.CreatedAt, &apiKey.UpdatedAt,
		&userIsActive,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("invalid or inactive API key")
		}
		return nil, err
	}

	// Additional safety check
	if !userIsActive {
		return nil, fmt.Errorf("user account is deactivated")
	}

	if err := json.Unmarshal(scopesJSON, &apiKey.Scopes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scopes: %w", err)
	}

	// Check if key is expired
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("API key expired")
	}

	return &apiKey, nil
}

func (r *Repository) UpdateAPIKeyUsage(ctx context.Context, keyID string) error {
	// Best-effort update, don't block on errors
	query := `
		UPDATE api_keys
		SET last_used_at = NOW(),
		    total_requests = total_requests + 1
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, keyID)
	return err
}

func (r *Repository) ListAPIKeysByUser(ctx context.Context, userID string) ([]*domain.APIKey, error) {
	query := `
		SELECT id, user_id, name, key_prefix, scopes, rate_limit_per_hour, 
		       is_active, expires_at, last_used_at, total_requests, created_at, updated_at
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apiKeys []*domain.APIKey
	for rows.Next() {
		var apiKey domain.APIKey
		var scopesJSON []byte

		err := rows.Scan(
			&apiKey.ID, &apiKey.UserID, &apiKey.Name, &apiKey.KeyPrefix,
			&scopesJSON, &apiKey.RateLimitPerHour, &apiKey.IsActive, &apiKey.ExpiresAt,
			&apiKey.LastUsedAt, &apiKey.TotalRequests, &apiKey.CreatedAt, &apiKey.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(scopesJSON, &apiKey.Scopes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scopes: %w", err)
		}

		apiKeys = append(apiKeys, &apiKey)
	}

	return apiKeys, nil
}

func (r *Repository) DeleteAPIKey(ctx context.Context, keyID, userID string) error {
	query := `
		DELETE FROM api_keys
		WHERE id = $1 AND user_id = $2
	`

	result, err := r.db.Exec(ctx, query, keyID, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

func (r *Repository) UpdateAPIKey(ctx context.Context, keyID, userID string, isActive bool) error {
	query := `
		UPDATE api_keys
		SET is_active = $3, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`

	result, err := r.db.Exec(ctx, query, keyID, userID, isActive)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

// Check if user exists and is active
func (r *Repository) IsUserActive(ctx context.Context, userID string) (bool, error) {
	query := `SELECT is_active FROM users WHERE user_id = $1`
	
	var isActive bool
	err := r.db.QueryRow(ctx, query, userID).Scan(&isActive)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, fmt.Errorf("user not found")
		}
		return false, err
	}

	return isActive, nil
}