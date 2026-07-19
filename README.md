# jcrawl

A multi-user Go service that monitors online availability and automatically books items based on user preferences. Currently supports restaurant reservations via Google Maps.

## Features

- **Multi-user support** - Multiple users with isolated preferences and bookings
- **Restaurant monitoring** - Check Google Maps restaurant availability
- **Background worker** - Checks availability every 5 minutes across all users
- **REST API** - Full API for user registration, preference management, and booking history
- **Database storage** - PostgreSQL for persistent, scalable data storage
- **Auto-booking** - Automatically books when availability matches preferences
- **Date/day filtering** - Filter by date range and day of week preferences

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    REST API                         │
│  (Register, Login, Preferences, Bookings)           │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
        ┌────────────────────────┐
        │   HTTP Server (8080)   │
        └────────────┬───────────┘
                     │
        ┌────────────┴───────────┐
        ▼                        ▼
  ┌─────────────┐         ┌──────────────┐
  │ PostgreSQL  │         │   Worker     │
  │ Database    │         │ (5min check) │
  └─────────────┘         └──────┬───────┘
                                 │
                                 ▼
                        ┌──────────────────┐
                        │ Restaurant Check │
                        │   + Auto-Book    │
                        └──────────────────┘
```

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 12+

### Installation

1. Clone the repository:
```bash
git clone https://github.com/javid-p84/jcrawl.git
cd jcrawl
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment:
```bash
cp .env.example .env
```

4. Configure PostgreSQL connection in `.env`:
```
DATABASE_URL=postgres://user:password@localhost:5432/jcrawl?sslmode=disable
```

5. Build:
```bash
go build -o jcrawl
```

6. Run:
```bash
./jcrawl
```

Server starts on `http://localhost:8080`

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Create new account
- `POST /api/v1/auth/login` - Login and get token

### Preferences
- `POST /api/v1/preferences` - Create monitoring preference
- `GET /api/v1/preferences` - List user's preferences

### Bookings
- `GET /api/v1/bookings` - List user's booking history

### Health
- `GET /health` - Service health check

## Example Usage

```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secure123"}'

# Add restaurant preference
curl -X POST http://localhost:8080/api/v1/preferences \
  -H "Content-Type: application/json" \
  -H "X-User-ID: <user-id>" \
  -d '{
    "google_link": "https://www.google.com/maps/...",
    "restaurant_name": "Restaurant Name",
    "date_range_from": "2024-01-01",
    "date_range_to": "2024-01-31",
    "day_preference": [5, 6],
    "party_size": 2
  }'

# Get preferences
curl http://localhost:8080/api/v1/preferences \
  -H "X-User-ID: <user-id>"
```

## Project Structure

- `pkg/models/` - Data structures (User, Preferences, Bookings)
- `pkg/db/` - Database layer (repositories, schema)
- `pkg/api/` - HTTP handlers and routes
- `pkg/restaurant/` - Restaurant availability checking
- `pkg/worker/` - Background job for availability checks
- `pkg/config/` - Configuration management

## How It Works

### 1. User Creates Preference
User provides:
- Google Maps link to restaurant
- Date range (e.g., Jan 1-31)
- Day preferences (e.g., Fri/Sat only)
- Party size (e.g., 2 people)
- **Guest details** (name, email, phone) for auto-booking
- Optional special notes (dietary restrictions, seating preferences)

### 2. Background Worker Monitors
Every 5 minutes, the worker:
- Fetches all active user preferences
- For each preference, checks each date matching day preferences
- Uses Chrome/Chromium to load the restaurant booking page
- Parses the HTML to extract available time slots
- Stores found availabilities in database

### 3. Parser Handles Multiple Platforms
Supports various booking platforms:
- **Resy** - Extracts from `data-time` attributes
- **OpenTable** - Parses availability buttons
- **Google Reserve** - Handles button-based slots
- **Generic** - Fallback pattern matching for times

### 4. Auto-Booking (Now Implemented!)
When availability is found and `auto_book: true`:
1. **Detect Platform** - Identifies which booking system (Resy, OpenTable, etc.)
2. **Fill Form** - Automatically fills:
   - Guest name
   - Email address
   - Phone number
   - Special notes/preferences
3. **Click Submit** - Completes the reservation
4. **Capture Confirmation** - Extracts confirmation ID
5. **Store Record** - Saves booking in database with status and confirmation ID
6. **Deactivate** - Stops monitoring after successful booking

## Scraping Technology

Uses **chromedp** for browser automation:
- Headless Chrome/Chromium
- Handles JavaScript-heavy booking pages
- Waits for dynamic content to load
- Extracts full page HTML

**Requirements:**
- Chrome or Chromium installed on system
- Linux: `sudo apt-get install chromium-browser`
- macOS: `brew install chromium` or use system Chrome
- Windows: Download from chromium.woolyss.com

## Booking Platforms Supported

| Platform | Format | Status |
|----------|--------|--------|
| **Resy** | data-time attributes, form inputs | ✅ Implemented |
| **OpenTable** | Button-based time slots | ✅ Implemented |
| **Google Reserve** | Dialog-based booking | ✅ Implemented |
| **Generic** | Text-based times + common inputs | ✅ Fallback |

## Example Workflow

```
1. User creates preference:
   POST /api/v1/preferences
   {
     "google_link": "https://www.google.com/maps/place/...",
     "restaurant_name": "Michelin Star Restaurant",
     "date_range_from": "2024-01-15",
     "date_range_to": "2024-02-15",
     "day_preference": [5, 6],  // Fri/Sat
     "party_size": 2,
     "auto_book": true,
     "guest_name": "John Doe",
     "guest_email": "john@example.com",
     "guest_phone": "+1234567890"
   }

2. Worker checks every 5 minutes:
   - Loads restaurant page
   - Finds: Jan 17 (Fri) at 7:30 PM available
   - Auto-books with guest info
   - Saves confirmation

3. User gets confirmation:
   - Email sent (future feature)
   - Booking record in database
   - Preference auto-deactivated
```

## Next Steps

- [ ] Implement JWT authentication with tokens
- [ ] Add email notifications on successful bookings
- [ ] Add Slack integration for notifications
- [ ] Support direct restaurant APIs (Yelp, TripAdvisor)
- [ ] Create web dashboard for preferences/bookings
- [ ] Add support for appointment-based bookings (dentist, doctor, salons)
- [ ] Implement retry logic for failed bookings
- [ ] Add user preferences for best time/party size combinations
