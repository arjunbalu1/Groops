package auth

// UserInfo represents the user information returned by Google's userinfo endpoint
type UserInfo struct {
	Sub           string `json:"sub"`            // Unique Google ID
	Email         string `json:"email"`          // User's email
	EmailVerified bool   `json:"email_verified"` // Whether the email is verified
	Name          string `json:"name"`           // Full name
	GivenName     string `json:"given_name"`     // First name
	FamilyName    string `json:"family_name"`    // Last name
	Picture       string `json:"picture"`        // Profile picture URL
	Locale        string `json:"locale"`         // User's locale/language
}
