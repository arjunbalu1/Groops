package auth

import (
	"fmt"
	"groops/internal/models"
	"time"

	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

// SaveRefreshTokenToAccount encrypts and saves a refresh token to the user's account
func SaveRefreshTokenToAccount(db *gorm.DB, googleID string, token *oauth2.Token) error {
	if token == nil || token.RefreshToken == "" {
		return nil // No refresh token to save
	}

	// Encrypt the refresh token
	encryptedToken, err := EncryptRefreshToken(token.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	// Find the account and update
	updates := map[string]interface{}{
		"encrypted_refresh_token": encryptedToken,
		"token_expiry":            token.Expiry,
	}

	if err := db.Model(&models.Account{}).
		Where("google_id = ?", googleID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to save refresh token to account: %w", err)
	}

	return nil
}

// GetRefreshTokenFromAccount retrieves and decrypts a refresh token from an account
func GetRefreshTokenFromAccount(db *gorm.DB, googleID string) (string, time.Time, error) {
	var account models.Account

	if err := db.Select("encrypted_refresh_token, token_expiry").
		Where("google_id = ?", googleID).
		First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", time.Time{}, fmt.Errorf("account not found")
		}
		return "", time.Time{}, fmt.Errorf("failed to retrieve account: %w", err)
	}

	// Decrypt the refresh token
	refreshToken, err := DecryptRefreshToken(account.EncryptedRefreshToken)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to decrypt refresh token: %w", err)
	}

	return refreshToken, account.TokenExpiry, nil
}

// UpdateAccountToken updates an account's token information
func UpdateAccountToken(db *gorm.DB, googleID string, token *oauth2.Token) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}

	updates := map[string]interface{}{
		"token_expiry": token.Expiry,
	}

	// If we got a new refresh token, encrypt and update it
	if token.RefreshToken != "" {
		encryptedToken, err := EncryptRefreshToken(token.RefreshToken)
		if err != nil {
			return fmt.Errorf("failed to encrypt refresh token: %w", err)
		}
		updates["encrypted_refresh_token"] = encryptedToken
	}

	if err := db.Model(&models.Account{}).
		Where("google_id = ?", googleID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update account token: %w", err)
	}

	return nil
}

// NeedsTokenRefresh checks if the account's token needs to be refreshed
func NeedsTokenRefresh(expiry time.Time) bool {
	// Refresh 5 minutes before expiry to avoid edge cases
	return time.Now().Add(time.Minute * 5).After(expiry)
}
