package domain

import "time"

// AuthContext represents the authenticated user/api customer
type AuthContext struct {
	UserID   string   `json:"user_id"`
	UserType string   `json:"user_type"` // "clerk" or "api_customer"
	Email    string   `json:"email,omitempty"`
	Name     string   `json:"name,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
}

// APIKey represents an API key for authentication
type APIKey struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	KeyHash          string    `json:"-"` // Never expose in JSON
	Name             string    `json:"name"`
	KeyPrefix        string    `json:"key_prefix"`
	Scopes           []string  `json:"scopes"`
	RateLimitPerHour int       `json:"rate_limit_per_hour"`
	IsActive         bool      `json:"is_active"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	LastUsedAt       *time.Time `json:"last_used_at,omitempty"`
	TotalRequests    int64     `json:"total_requests"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// User represents a user in the system (Clerk or API customer)
type User struct {
	UserID      string    `json:"user_id"`
	UserType    string    `json:"user_type"`
	Email       string    `json:"email,omitempty"`
	Name        string    `json:"name,omitempty"`
	ClerkUserID *string   `json:"clerk_user_id,omitempty"`
	CompanyName *string   `json:"company_name,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateAPIKeyRequest for creating new API keys
type CreateAPIKeyRequest struct {
	Name             string    `json:"name"`
	Scopes           []string  `json:"scopes,omitempty"`
	RateLimitPerHour int       `json:"rate_limit_per_hour,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKeyResponse includes the raw key (only shown once)
type CreateAPIKeyResponse struct {
	APIKey
	RawKey string `json:"raw_key"` // Only returned on creation
}

// ClerkJWTClaims represents the claims from Clerk JWT
type ClerkJWTClaims struct {
	Sub   string `json:"sub"`   // Clerk user ID
	Email string `json:"email"`
	Name  string `json:"name"`
	Exp   int64  `json:"exp"`
	Iat   int64  `json:"iat"`
}