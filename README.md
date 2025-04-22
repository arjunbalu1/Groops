# Groops

A group management platform where people can host and join groups with location-based features.
pls refer to the sketch pdf lol

## UNDER CONSTRUCTION

## Technologies Used

- Go (Golang)
- Gin Web Framework
- GORM ORM
- PostgreSQL
- JWT Authentication with secure cookies
- RESTful API design

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

The server will start on the port specified in your .env file (defaults to 8080).

## API Endpoints

### Health Check
- `GET /health` - Check if the server is running
- `GET /` - Welcome message

### Authentication
- `POST /accounts` - Create a new account
- `POST /auth/login` - Login with username and password
- `POST /auth/refresh` - Refresh access token using refresh token

### Protected Routes (requires authentication)
- `GET /api/accounts/:username` - Get account details
- `POST /api/groups` - Create a new group
- `GET /api/groups` - List all groups

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

## Security Implementation

- **JWT Authentication**: Secure authentication using JWT tokens stored in HttpOnly cookies
- **SameSite=Strict**: Cookies are protected against CSRF attacks
- **Secure Flag**: Cookies are only sent over HTTPS connections
- **Path Restriction**: Access tokens limited to API routes, refresh tokens to auth endpoint

## Environment Variables

Key environment variables:
- `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT`: Database configuration
- `PORT`: Server port (defaults to 8080)
- `JWT_SECRET`: Secret key for signing JWT tokens
- `APP_ENV`: Application environment (development/production)

## Development Notes

- Password hashing is implemented using bcrypt
- Environment variables are strictly validated at startup
- Redis caching is configured but not currently used

