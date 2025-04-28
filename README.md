# Groops

A group management platform where people can host and join activity-based groups with location, filtering, and social features.

## Table of Contents

- [Features](#features)
- [Technologies Used](#technologies-used)
- [Setup Instructions](#setup-instructions)
- [Authentication & Session Flow](#authentication--session-flow)
- [API Endpoints](#api-endpoints)
- [Data Models](#data-models)
- [Database Reset](#database-reset)
- [Postman & API Demo Guide](#postman--api-demo-guide)
- [Troubleshooting](#troubleshooting)
- [Environment Variables](#environment-variables)
- [Development Notes](#development-notes)

---

## Features

- **Google OAuth Authentication**: Secure login with Google, no passwords required
- **Profile Management**: Users can update their bio and avatar
- **Group Creation & Management**: Create, join, leave, and manage groups for different activities
- **Filtering, Sorting, Pagination**: Find groups by activity, skill, price, date, and more
- **Activity Tracking**: Track user participation and group activity
- **Location-Based Features**: Store and search for groups by venue
- **Enhanced Security**: Secure cookies, input validation, and CSRF protection

---

## Technologies Used

- Go (Golang)
- Gin Web Framework
- GORM ORM
- PostgreSQL
- Google OAuth2
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
    - **Option 1: Terminal**
      ```sh
      createdb -U postgres groops
      ```
    - **Option 2: pgAdmin**
      - Open pgAdmin, right-click "Databases" → Create → Database, name it `groops`.

4. Run the server:
    ```sh
    go run cmd/server/main.go
    ```
    The server will start on the port specified in your `.env` file (defaults to 8080).

---

## Authentication & Session Flow

- **Login:** Users authenticate via Google OAuth (`/auth/login`).
- **Profile Creation:** On first login, users are prompted to create a profile (choose a username, set bio/avatar).
- **Session:** After profile creation, all requests are authenticated via a secure session cookie (`groops_session`).
- **Re-login:** If your session expires, logging in again with Google will automatically link your session to your existing profile (by Google ID).

---

## API Endpoints

### Health & Info
- `GET /health` — Check if the server is running
- `GET /` — Welcome message

### Authentication
- `GET /auth/login` — Start Google OAuth login
- `GET /auth/logout` — Logout and clear session

### Account/Profile
- `GET /api/accounts/:username` — Get account details
- `PUT /api/accounts/:username` — Update your profile (bio, avatar)
- `POST /api/profile/register` — Register your profile after OAuth login

### Groups
- `POST /api/groups` — Create a new group (no organizer_username in payload; backend uses your session)
- `GET /api/groups` — List groups (supports filtering, sorting, pagination)
- `POST /api/groups/:group_id/join` — Request to join a group
- `POST /api/groups/:group_id/leave` — Leave a group
- `GET /api/groups/:group_id/pending-members` — List pending join requests (organiser only)
- `POST /api/groups/:group_id/members/:username/approve` — Approve join request
- `POST /api/groups/:group_id/members/:username/reject` — Reject join request

### Notifications
- `GET /api/notifications` — List notifications
- `GET /api/notifications/unread-count` — Get unread notification count

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

(See `internal/models/` for full struct definitions)

### Account
- `Username` (Primary Key)
- `Email`
- `GoogleID`
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

## Database Reset

**To reset your database (development):**

- **Terminal:**
  ```sh
  psql -U postgres -h localhost -c "DROP DATABASE IF EXISTS groops;"
  psql -U postgres -h localhost -c "CREATE DATABASE groops;"
  ```
- **pgAdmin:**
  - Right-click the `groops` database → Delete/Drop
  - Right-click "Databases" → Create → Database, name it `groops`

After resetting, restart your Go server to re-run migrations and recreate all tables.

---

## Postman & API Demo Guide

- See `API_DEMO_GUIDE.md` for a step-by-step guide to using all endpoints with Postman, including authentication and session cookie handling.
- All authenticated requests require the `groops_session` cookie (see the guide for details).

---

## Troubleshooting

- **Database errors:** Ensure your database exists and credentials in `.env` are correct.
- **Migrations:** If tables are missing, reset the DB and restart the server.
- **Session/cookie issues:** If you get `authentication required`, repeat the OAuth login and use the new session cookie.
- **Group creation:** Do not include `organizer_username` in the payload; the backend uses your session.
- **Stale dependencies:** Run `go get -u ./...` and `go mod tidy` to update dependencies.

---

## Environment Variables

- `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT`: Database config
- `PORT`: Server port (default 8080)
- `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL`: Google OAuth config
- `APP_ENV`: Application environment (development/production)

---

## Development Notes

- Passwords are not used; all authentication is via Google OAuth.
- Strict environment variable validation at startup.
- See `sketch.pdf` for UI/UX ideas.

---

If you have any questions or want to contribute, please open an issue or pull request!
