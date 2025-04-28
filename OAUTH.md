# Google OAuth Implementation Guide

This document explains the Google OAuth integration in Groops.

## Overview

Groops uses Google OAuth 2.0 for authentication, following industry-standard best practices:

1. **Authorization Code Flow** - Secure server-side flow
2. **Session Management** - Using server-side sessions with secure cookies
3. **State Parameter** - CSRF protection with cryptographically secure random strings
4. **Token Refresh** - Automatic refresh when access tokens expire
5. **Database Storage** - Secure storage of tokens and sessions

## Flow Diagram

```
┌─────────┐          ┌─────────┐          ┌──────────┐
│  User   │          │  Groops │          │  Google  │
└────┬────┘          └────┬────┘          └────┬─────┘
     │                    │                    │
     │ 1. Click Login     │                    │
     │ ──────────────────>│                    │
     │                    │                    │
     │ 2. Redirect to     │                    │
     │    Google + State  │                    │
     │ <────────────────  │                    │
     │                    │                    │
     │ 3. Google Auth     │                    │
     │ ──────────────────────────────────────> │
     │                    │                    │
     │ 4. Redirect with   │                    │
     │    Code + State    │                    │
     │ <────────────────────────────────────── │
     │                    │                    │
     │ 5. Exchange Code   │                    │
     │                    │ ─────────────────> │
     │                    │                    │
     │ 6. Tokens          │                    │
     │                    │ <───────────────── │
     │                    │                    │
     │ 7. Create Session  │                    │
     │    Set Cookie      │                    │
     │ <────────────────  │                    │
     │                    │                    │
```

## Technical Implementation

### 1. Configuration

The OAuth configuration uses environment variables:

```
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback
```

### 2. Security Features

#### CSRF Protection with State Parameter

Every login request generates a cryptographically secure random state:

```go
// Generate and store a secure random state
state, err := SetOAuthState(c)
if err != nil {
    return "", err
}
// Use state in authorization URL
return googleOAuthConfig.AuthCodeURL(state), nil
```

The state is stored in a secure, HttpOnly cookie and verified on callback.

#### Secure Session Storage

Sessions are stored in the database with:
- Random session IDs
- HttpOnly cookies (inaccessible to JavaScript)
- Secure flag (HTTPS only in production)
- Automatic expiration

#### Token Security

Access and refresh tokens are:
- Never exposed to the client
- Stored securely in the database
- Refreshed automatically before expiry

### 3. Session Management

Sessions are managed through the `Session` model:

```go
type Session struct {
    ID           string    // Random unique ID
    UserID       string    // Google ID
    Username     string    // Optional username
    AccessToken  string    // OAuth access token
    RefreshToken string    // OAuth refresh token
    TokenExpiry  time.Time // When token expires
    CreatedAt    time.Time // When session was created
    ExpiresAt    time.Time // When session expires
}
```

### 4. Code Flow Documentation

#### Login Process

1. User visits `/auth/login`
2. Server generates secure random state and stores in cookie
3. Server redirects to Google with state parameter
4. User authenticates with Google
5. Google redirects back with code and state
6. Server verifies state to prevent CSRF
7. Server exchanges code for tokens
8. Server creates session and sets session cookie
9. Server redirects to dashboard or profile creation

#### Authentication Middleware

```go
// AuthMiddleware validates sessions and refreshes tokens
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get session from cookie
        session, err := GetSession(c)
        if err != nil {
            // Redirect to login
            return
        }
        
        // Refresh token if needed
        if session.NeedsTokenRefresh() {
            RefreshSessionToken(c, session)
        }
        
        // Set user info in context
        c.Set("sub", session.UserID)
        c.Set("username", session.Username)
        
        c.Next()
    }
}
```

## Redis Integration (Future)

This implementation is designed for easy migration to Redis:

1. The Session model can be serialized to JSON
2. The database access is isolated in specific functions
3. Session IDs are used as keys

To migrate to Redis:
- Replace database calls with Redis SET/GET operations
- Use Redis TTL for expiration
- Keep the same API interface

## Common OAuth Issues and Solutions

### 1. Token Expiry

Google access tokens expire after 1 hour. Our solution:
- Store the expiry time in the session
- Check before each authenticated request
- Refresh automatically using refresh token

### 2. Session Security

Sessions could be hijacked if cookies are stolen. Our protections:
- HttpOnly cookies (no JavaScript access)
- Secure flag (HTTPS only in production)
- Session validation on each request

### 3. CSRF Attacks

Cross-Site Request Forgery attacks are prevented by:
- Random state parameter in the OAuth flow
- Verifying state parameter on callback

## Testing OAuth

To test the OAuth flow:

1. Start the server: `go run cmd/server/main.go`
2. Visit http://localhost:8080/auth/login
3. Authenticate with Google
4. You'll be redirected back to Groops

## References

- [Google OAuth Documentation](https://developers.google.com/identity/protocols/oauth2/web-server)
- [OAuth 2.0 Security Best Practices](https://oauth.net/2/best-practices/)
- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html) 
hehe