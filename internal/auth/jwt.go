package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// TokenClaims represents the claims in the JWT token
type TokenClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT token for a user
func GenerateToken(username string) (string, error) {
	// Get secret key from environment
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		return "", fmt.Errorf("JWT_SECRET environment variable not set")
	}

	// Get expiry duration from environment
	expiryStr := os.Getenv("JWT_EXPIRY")
	if expiryStr == "" {
		expiryStr = "24h" // Default to 24 hours
	}

	// Parse expiry duration
	expiry, err := time.ParseDuration(expiryStr)
	if err != nil {
		return "", fmt.Errorf("invalid JWT_EXPIRY format: %w", err)
	}

	// Create token with claims
	claims := TokenClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "groops",
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates and parses a JWT token
func ValidateToken(tokenString string) (*TokenClaims, error) {
	// Get secret key from environment
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable not set")
	}

	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
