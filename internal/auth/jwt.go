package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// TokenType defines the type of token being generated
type TokenType string

const (
	// AccessToken is a short-lived token used for authenticated requests
	AccessToken TokenType = "access"
	// RefreshToken is a longer-lived token used to obtain new access tokens
	RefreshToken TokenType = "refresh"
)

// Cookie configurations
const (
	AccessTokenCookieName  = "access_token"
	RefreshTokenCookieName = "refresh_token"
	AccessTokenExpiry      = 15 * time.Minute
	RefreshTokenExpiry     = 24 * time.Hour
)

// CustomClaims represents the claims in our JWT tokens
type CustomClaims struct {
	jwt.RegisteredClaims
	Username  string    `json:"username"`
	TokenType TokenType `json:"token_type"`
}

// GetSecretKey returns the JWT secret key from environment variables
func GetSecretKey() (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET environment variable not set")
	}
	return secret, nil
}

// GenerateToken creates a new JWT token
func GenerateToken(username string, tokenType TokenType) (string, time.Time, error) {
	// Get the secret key
	secretKey, err := GetSecretKey()
	if err != nil {
		return "", time.Time{}, err
	}

	// Define token expiration based on type
	var expirationTime time.Time
	if tokenType == AccessToken {
		expirationTime = time.Now().Add(AccessTokenExpiry)
	} else {
		expirationTime = time.Now().Add(RefreshTokenExpiry)
	}

	// Create the claims
	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "groops-api",
			Subject:   username,
		},
		Username:  username,
		TokenType: tokenType,
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expirationTime, nil
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*CustomClaims, error) {
	// Get the secret key
	secretKey, err := GetSecretKey()
	if err != nil {
		return nil, err
	}

	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	// Validate and extract claims
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
