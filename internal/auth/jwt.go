package auth

import (
	"fmt"
	"os"
	"time"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	Username     string `json:"username"`
	TokenVersion int    `json:"token_version"`
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT token with version for invalidation
func GenerateToken(username string, tokenVersion int) (string, error) {
	// Get JWT settings from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return "", fmt.Errorf("JWT_SECRET not set in environment")
	}

	// 24 hour expiry by default
	expiry := 24 * time.Hour

	claims := JWTClaims{
		Username:     username,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "groops-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*JWTClaims, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET not set in environment")
	}

	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// SetAuthCookie sets the JWT token as a cookie with security settings
func SetAuthCookie(c *gin.Context, username string, tokenVersion int) error {
	// Generate token
	token, err := GenerateToken(username, tokenVersion)
	if err != nil {
		return err
	}

	// Set same site mode to Strict
	c.SetSameSite(http.SameSiteStrictMode)

	// Get the domain from environment or use a default
	domain := os.Getenv("COOKIE_DOMAIN")

	// Set auth cookie with security settings
	c.SetCookie(
		"auth_token", // Name
		token,        // Value
		24*60*60,     // Max age: 24 hours in seconds
		"/",          // Path
		domain,       // Domain: set from environment
		false,        // Secure: set to true in production (HTTPS only)
		true,         // HttpOnly: prevents JavaScript access
	)

	return nil
}

// ClearAuthCookie removes the authentication cookie
func ClearAuthCookie(c *gin.Context) {
	// Get the domain from environment or use empty string
	domain := os.Getenv("COOKIE_DOMAIN")

	// Set same site mode to Strict
	c.SetSameSite(http.SameSiteStrictMode)

	// Clear the cookie
	c.SetCookie("auth_token", "", -1, "/", domain, false, true)
}

// GetUsernameFromContext gets the authenticated username from the context
func GetUsernameFromContext(c *gin.Context) string {
	username, exists := c.Get("username")
	if !exists {
		return ""
	}
	return username.(string)
}
