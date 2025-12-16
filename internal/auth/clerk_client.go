package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ClerkClient struct {
	secretKey  string
	baseURL    string
	httpClient *http.Client
}

type ClerkUser struct {
	ID            string `json:"id"`
	EmailAddress  string `json:"primary_email_address_id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	EmailAddresses []struct {
		EmailAddress string `json:"email_address"`
		ID           string `json:"id"`
	} `json:"email_addresses"`
}

func NewClerkClient(secretKey string) *ClerkClient {
	return &ClerkClient{
		secretKey:  secretKey,
		baseURL:    "https://api.clerk.com/v1",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetUser fetches user details from Clerk Backend API
func (c *ClerkClient) GetUser(ctx context.Context, userID string) (*ClerkUser, error) {
	url := fmt.Sprintf("%s/users/%s", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user from Clerk: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Clerk API error: status %d", resp.StatusCode)
	}

	var user ClerkUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode Clerk response: %w", err)
	}

	return &user, nil
}

// GetUserEmail extracts the primary email from Clerk user
func (u *ClerkUser) GetEmail() string {
	if len(u.EmailAddresses) > 0 {
		// Find primary email
		for _, email := range u.EmailAddresses {
			if email.ID == u.EmailAddress {
				return email.EmailAddress
			}
		}
		// Fallback to first email
		return u.EmailAddresses[0].EmailAddress
	}
	return ""
}

// GetFullName combines first and last name
func (u *ClerkUser) GetFullName() string {
	firstName := strings.TrimSpace(u.FirstName)
	lastName := strings.TrimSpace(u.LastName)

	if firstName != "" && lastName != "" {
		return firstName + " " + lastName
	}
	if firstName != "" {
		return firstName
	}
	if lastName != "" {
		return lastName
	}
	return ""
}