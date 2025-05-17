# Groops 🌍
hehe
**The app is live at [https://groops.fun](https://groops.fun)**

All API calls should use the base URL: `https://groops.fun`

Adjust any API requests accordingly. For example:

```
curl -X POST https://groops.fun/api/profile/register ...
```

> A social platform for activity-based group creation and participation

Groops is a powerful backend API (currently) that enables users to create, join, and manage activity-based groups with location services, advanced filtering, and social features.

## 📋 Features

- **Streamlined Authentication**: Google OAuth integration with secure session management
- **Rich Profile System**: Customizable user profiles with bio and avatar
- **Group Management**: Create, join, leave, and manage groups for various activities
- **Advanced Filtering**: Find groups by activity type, skill level, price, date, and more
- **Activity Tracking**: Comprehensive history of user participation
- **Location Services**: Geographic search and venue management with Google Maps integration
- **Notifications**: Real-time notification system for group events

## 🛠️ Technologies

- **Backend**: Go, Gin Web Framework
- **Database**: PostgreSQL with GORM ORM
- **Authentication**: Google OAuth 2.0
- **Maps Integration**: Google Maps Places API
- **API Design**: RESTful architecture with JSON

## 🚀 Getting Started

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

2. Configure environment variables:
   ```sh
   cp .env.example .env
   # Edit .env with your database and OAuth credentials
   ```

3. Set up the database:
   ```sh
   createdb -U postgres groops
   ```

4. Start the server:
   ```sh
   go run cmd/server/main.go
   ```

## 🔌 API Reference

### Auth Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/auth/login` | Start Google OAuth flow |
| GET | `/auth/logout` | Clear session |

### Profile Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/accounts/:username` | Get user profile |
| PUT | `/api/profile` | Update your profile (requires auth) |
| POST | `/api/profile/register` | Create profile after OAuth login |

### Group Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/groups` | Create a new group |
| GET | `/api/groups` | List groups (with filters) |
| POST | `/api/groups/:group_id/join` | Request to join a group |
| POST | `/api/groups/:group_id/leave` | Leave a group |
| GET | `/api/groups/:group_id/pending-members` | List pending join requests |
| POST | `/api/groups/:group_id/members/:username/approve` | Approve join request |
| POST | `/api/groups/:group_id/members/:username/reject` | Reject join request |

### Location Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/locations/validate?place_id=PLACE_ID` | Validate and standardize a Google Place ID |

### Notification Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/notifications` | List your notifications |
| GET | `/api/notifications/unread-count` | Get unread count |

## 📧 Email Notifications

The platform sends email notifications using SendGrid for important events:

- **Join Requests**: Group organizers receive an email when someone requests to join their group
- **Join Approvals**: Users receive an email when their join request is approved
- **Event Reminders**: Group members receive reminders 24 hours and 1 hour before events

Email notifications are handled automatically by the system using a background worker that checks for upcoming events every 5 minutes.

## 🔍 Group Filtering

The `/api/groups` endpoint supports powerful filtering with query parameters:

- `activity_type` - Filter by activity category (sport, social, games, other)
- `skill_level` - Filter by required skill (beginner, intermediate, advanced)
- `min_price`/`max_price` - Price range filtering
- `date_from`/`date_to` - Date range filtering
- `organiser_id` - Filter by group creator
- `sort_by`/`sort_order` - Control result ordering
- `limit`/`offset` - Pagination controls

Example: `/api/groups?activity_type=sport&skill_level=beginner&sort_by=date_time&sort_order=asc`

## 🔒 Authentication Flow

1. **Login**: User authenticates via Google OAuth (`/auth/login`)
2. **Profile Creation**: First-time users create a profile with username, bio, and avatar
3. **Session**: A secure cookie (`groops_session`) authenticates all future requests
4. **Auto-linking**: Returning users' sessions are automatically linked to their existing profiles

## 🗺️ Location Services

The platform integrates with Google Maps Places API to provide reliable and standardized location data:

1. **Place ID Validation**: Validate Google Place IDs via `/api/locations/validate` endpoint
2. **Standardized Location Data**: Store consistent location data including:
   - Place ID (Google's unique identifier)
   - Formatted address
   - Place name
   - Precise coordinates (latitude/longitude)
3. **Location Storage**: Locations are stored as JSONB objects in the database with groups
4. **Integration Flow**:
   - Frontend sends a Place ID to validate
   - Backend verifies and standardizes the location data
   - Standardized location is then used with group creation/updates

To implement the frontend integration:    (TODO:FRONTEND)
1. Load the Google Maps JavaScript API with Places library
2. Implement autocomplete search for locations
3. Extract the Place ID from selected places
4. Validate the Place ID with the backend
5. Use the standardized location data for group creation

## 🧰 Troubleshooting

- **Auth errors**: If you see `authentication required`, your session may have expired - login again
- **DB issues**: Ensure your PostgreSQL server is running and credentials are correct
- **Profile updates**: Use the `/api/profile` endpoint which uses your session identity
- **Dependencies**: Run `go mod tidy` to ensure all dependencies are up to date

## 🌐 Environment Variables

Configure the following in your `.env` file:

```
# Database
DB_HOST=localhost
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=groops
DB_PORT=5432

# Server
PORT=8080

# OAuth
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback

# Google Maps API
GOOGLE_MAPS_API_KEY=your-maps-api-key

```

To set up Google Maps:
1. Create a project in Google Cloud Console
2. Enable the Maps JavaScript API and Places API
3. Create an API key with appropriate restrictions
4. Add the key to your `.env` file

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. 
