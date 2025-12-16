package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ClerkVerifier struct {
	jwksURL string
	keys    map[string]*rsa.PublicKey
	client  *http.Client
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewClerkVerifier(jwksURL string) *ClerkVerifier {
	return &ClerkVerifier{
		jwksURL: jwksURL,
		keys:    make(map[string]*rsa.PublicKey),
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (cv *ClerkVerifier) VerifyToken(tokenString string) (*ClerkClaims, error) {
	// Parse the token without verification first to get the kid
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &ClerkClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Get the key ID from token header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("token missing kid in header")
	}

	// Get the public key
	publicKey, err := cv.getPublicKey(kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Verify and parse the token
	parsedToken, err := jwt.ParseWithClaims(tokenString, &ClerkClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	if !parsedToken.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := parsedToken.Claims.(*ClerkClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

func (cv *ClerkVerifier) getPublicKey(kid string) (*rsa.PublicKey, error) {
	// Check if we already have this key cached
	if key, exists := cv.keys[kid]; exists {
		return key, nil
	}

	// Fetch JWKS
	if err := cv.fetchJWKS(); err != nil {
		return nil, err
	}

	// Try again after fetching
	if key, exists := cv.keys[kid]; exists {
		return key, nil
	}

	return nil, fmt.Errorf("key with kid %s not found", kid)
}

func (cv *ClerkVerifier) fetchJWKS() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", cv.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := cv.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch JWKS: status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Convert JWKs to RSA public keys
	for _, jwk := range jwks.Keys {
		publicKey, err := jwkToRSAPublicKey(jwk)
		if err != nil {
			continue // Skip invalid keys
		}
		cv.keys[jwk.Kid] = publicKey
	}

	return nil
}

func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode the modulus
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode the exponent
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert bytes to big.Int
	n := new(big.Int).SetBytes(nBytes)
	
	// Convert exponent bytes to int
	var e int
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

type ClerkClaims struct {
	jwt.RegisteredClaims
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (c *ClerkClaims) GetUserID() string {
	return c.Subject
}

func (c *ClerkClaims) GetEmail() string {
	return c.Email
}

func (c *ClerkClaims) GetName() string {
	name := strings.TrimSpace(c.FirstName + " " + c.LastName)
	if name == "" {
		return c.Email
	}
	return name
}