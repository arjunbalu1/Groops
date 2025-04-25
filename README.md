# Groops

A group management platform where people can host and join activity-based groups with location, filtering, and social features.

## Table of Contents

- [Features](#features)
- [Technologies Used](#technologies-used)
- [Setup Instructions](#setup-instructions)
- [API Endpoints](#api-endpoints)
- [Data Models](#data-models)
- [Security Implementation](#security-implementation)
- [Data Validation](#data-validation)
- [Error Handling](#error-handling)
- [Performance Optimizations](#performance-optimizations)
- [Environment Variables](#environment-variables)
- [Development Notes](#development-notes)

---

## Features

- **User Authentication**: Secure registration and login with JWT tokens (HttpOnly cookies)
- **Profile Management**: Users can update their bio and avatar
- **Group Creation & Management**: Create, join, leave, and manage groups for different activities
- **Filtering, Sorting, Pagination**: Find groups by activity, skill, price, date, and more
- **Activity Tracking**: Track user participation and group activity
- **Location-Based Features**: Store and search for groups by venue
- **Enhanced Security**: Password strength enforcement, secure cookies, and input validation

---

## Technologies Used

- Go (Golang)
- Gin Web Framework
- GORM ORM
- PostgreSQL
- JWT Authentication
- RESTful API design

---

## Setup Instructions

### Prerequisites

- Go 1.19+
- PostgreSQL
- Git

### Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/arjunbalu1/Groops.git
    cd Groops
    ```

2. Copy the environment variables example file and update with your configuration:
    ```sh
    cp .env.example .env
    ```

3. Set up the PostgreSQL database:
    ```sh
    createdb -U postgres groops
    ```

4. Run the server:
    ```sh
    go run cmd/server/main.go
    ```
    The server will start on the port specified in your `.env` file (defaults to 8080).

---

## API Endpoints

### Health & Info
- `GET /health` — Check if the server is running
- `GET /` — Welcome message

### Authentication
- `POST /accounts` — Create a new account
- `POST /auth/login` — Login with username and password
- `POST /auth/refresh` — Refresh access token using refresh token

### Account/Profile
- `GET /api/accounts/:username` — Get account details
- `PUT /api/accounts/:username` — Update your profile (bio, avatar)

### Groups
- `POST /api/groups` — Create a new group
- `GET /api/groups` — List groups (supports filtering, sorting, pagination)
- `POST /api/groups/:group_id/join` — Request to join a group
- `POST /api/groups/:group_id/leave` — Leave a group
- `GET /api/groups/:group_id/pending-members` — List pending join requests (organiser only)
- `POST /api/groups/:group_id/members/:username/approve` — Approve join request
- `POST /api/groups/:group_id/members/:username/reject` — Reject join request

---

## Filtering, Sorting, and Pagination

The `/api/groups` endpoint supports the following query parameters:

| Parameter      | Type     | Description / Example Value                |
|----------------|----------|-------------------------------------------|
| activity_type  | string   | sport, social, games, other               |
| skill_level    | string   | beginner, intermediate, advanced          |
| min_price      | float    | Minimum cost per person                   |
| max_price      | float    | Maximum cost per person                   |
| date_from      | date     | Only events after this date (YYYY-MM-DD)  |
| date_to        | date     | Only events before this date              |
| organiser_id   | string   | Filter by organiser username              |
| min_members    | int      | Groups with at least this many members    |
| max_members    | int      | Groups with at most this many members     |
| name           | string   | Search by group name (partial match)      |
| sort_by        | string   | Field to sort by (e.g., date_time, cost)  |
| sort_order     | string   | asc or desc                              |
| limit          | int      | Results per page (default 10, max 100)    |
| offset         | int      | Pagination offset (default 0)             |

**Example:**
```
GET /api/groups?activity_type=sport&skill_level=beginner&sort_by=cost&sort_order=desc&limit=5&offset=0
```

---

## Data Models

### Account
- `Username` (Primary Key)
- `Email`
- `HashedPass`
- `DateJoined`
- `Rating`
- `Bio`
- `AvatarURL`
- `Activities` (ActivityLog)
- `OwnedGroups` (Group)
- `JoinedGroups` (GroupMember)
- `LastLogin`
- `CreatedAt`
- `UpdatedAt`
- `DeletedAt`

### Group
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
- `Members` (GroupMember)
- `CreatedAt`
- `UpdatedAt`
- `DeletedAt`

### GroupMember
- `GroupID` (Primary Key, Foreign Key)
- `Username` (Primary Key, Foreign Key)
- `Status` (pending, approved, rejected)
- `JoinedAt`
- `UpdatedAt`

### ActivityLog
- `ID` (Primary Key)
- `Username` (Foreign Key)
- `EventType` (create_group, join_group, etc.)
- `GroupID`
- `Timestamp`

---

## Security Implementation

- **JWT Authentication**: Secure authentication using JWT tokens in HttpOnly cookies
- **SameSite=Strict**: Cookies protected against CSRF
- **Secure Flag**: Cookies sent only over HTTPS
- **Path Restriction**: Access tokens limited to API routes, refresh tokens to auth endpoint
- **Password Requirements**: Minimum 8 characters, at least one letter and one number
- **Error Handling**: Consistent, user-friendly error messages
- **Input Validation**: Comprehensive validation for all user inputs

---

## Data Validation

- **Group Creation**:
  - Future date validation for events
  - Cost can be zero or more
  - Description length limits
  - Activity type and skill level validation

- **User Registration**:
  - Username format (alphanumeric, 3-30 chars)
  - Email format
  - Password strength
  - Duplicate username/email detection

---

## Error Handling

- Centralized error handling with clear messages
- Logs detailed error information for debugging
- Handles common error types (not found, validation, etc.)
- Returns appropriate HTTP status codes

---

## Performance Optimizations

- **Database Indexes**: On frequently queried fields
- **Reliable Activity Logging**: Retry logic for activity logs
- **GORM Optimizations**: Prepared statement caching, connection pooling

---

## Environment Variables

- `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT`: Database config
- `PORT`: Server port (default 8080)
- `JWT_SECRET`: Secret key for JWT tokens
- `APP_ENV`: Application environment (development/production)

---

## Development Notes

- Passwords hashed with bcrypt
- Strict environment variable validation at startup
- Redis can be added for caching or real-time features (not required for MVP)
- See `sketch.pdf` for UI/UX ideas

---

If you have any questions or want to contribute, please open an issue or pull request!

