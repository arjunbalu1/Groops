# Groops

A group management platform where people can host and join groups with location-based features.

## UNDER CONSTRUCTION

## Technologies Used

- Go (Golang)
- Gin Web Framework
- GORM ORM
- PostgreSQL
- RESTful API design
- JWT Authentication with secure cookies

## Setup Instructions

### Prerequisites

- Go 1.19+
- PostgreSQL
- Git

### Installation

1. Clone the repository:
```
git clone https://github.com/arjunbalu1/Groops.git
cd Groops
```

2. Copy the environment variables example file and update with your configuration:
```
cp .env.example .env
```

3. Set up the PostgreSQL database:
```
createdb -U postgres groops
```

4. Run the server:
```
go run cmd/server/main.go
```

The server will start on port 8080 by default.

## API Endpoints

### Health Check
- `GET /health` - Check if the server is running
- `GET /` - Welcome message

### Authentication
- `POST /auth/login` - Login with username and password
- `POST /auth/logout` - Logout (protected route)
- `GET /auth/me` - Get current user information (protected route)

### Accounts
- `POST /accounts` - Create a new account
- `GET /accounts/:username` - Get account details

### Groups
- `POST /groups` - Create a new group (protected route)
- `GET /groups` - List all groups (protected route)
- `GET /public/groups` - List all groups (public route)

## Authentication System

Groops uses a secure JWT-based authentication system with the following features:

### Security Features
- **HTTP-only Cookies**: Prevents JavaScript access to tokens (XSS protection)
- **Domain Restriction**: Cookies are restricted to the specified domain
- **SameSite Strict**: Prevents CSRF attacks by restricting cross-origin cookies
- **Token Versioning**: Enables immediate invalidation of tokens on logout
- **Secure Flag** (Production only): Ensures cookies are only sent over HTTPS

### Authentication Flow
1. User registers via `/accounts` endpoint
2. User logs in via `/auth/login` endpoint
3. Server issues a JWT token stored in an HTTP-only cookie
4. Protected routes check for valid token with matching version
5. On logout, server increments token version, invalidating all existing tokens

### Environment Configuration
Update these settings in your `.env` file:
- `JWT_SECRET`: Secret key for signing JWT tokens
- `COOKIE_DOMAIN`: Domain restriction for cookies (use your actual domain in production)

## Data Models

### Account
The account model represents a user in the system with the following fields:
- `Username` (Primary Key)
- `Email`
- `HashedPass`
- `DateJoined`
- `Rating`
- `Activities` (Relationship to ActivityLog)
- `OwnedGroups` (Relationship to Group)
- `JoinedGroups` (Relationship to GroupMember)
- `LastLogin`
- `TokenVersion` (Used for JWT token invalidation)
- `CreatedAt`
- `UpdatedAt`
- `DeletedAt`

### Group
The group model represents an activity group with the following fields:
- `ID` (Primary Key)
- `Name`
- `DateTime`
- `Venue` (JSONB)
- `Cost`
- `SkillLevel`
- `ActivityType`
- `MaxMembers`
- `Description`
- `OrganiserID`
- `Members` (Relationship to GroupMember)
- `CreatedAt`
- `UpdatedAt`
- `DeletedAt`

### GroupMember
Represents a user's membership in a group:
- `GroupID` (Primary Key, Foreign Key to Group)
- `Username` (Primary Key, Foreign Key to Account)
- `Status` (pending, approved, rejected)
- `JoinedAt`
- `UpdatedAt`

### ActivityLog
Tracks user activity:
- `ID` (Primary Key)
- `Username` (Foreign Key to Account)
- `EventType` (create_group, join_group, etc.)
- `GroupID`
- `Timestamp`

## Development Notes

- Password hashing is currently not implemented for development purposes.
- For production deployment, enable the Secure flag for cookies and use HTTPS.
- Redis caching is configured but not currently used.

