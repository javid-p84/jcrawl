# jcrawl 🎯

**Automated booking service** that monitors availability and books campsites, restaurants, and more based on your preferences.

## ⚡ Quick Start (3 Steps)

### Option 1: Docker (Recommended - Easiest)

**Requirements:** Docker & Docker Compose

```bash
# 1. Clone the repository
git clone https://github.com/javid-p84/jcrawl.git
cd jcrawl

# 2. Start the app
docker-compose up

# 3. Visit http://localhost:8080
```

✅ **That's it!** Database initializes automatically.

### Option 2: Local Development

**Requirements:** Go 1.21+, PostgreSQL 12+, Chrome/Chromium

```bash
# 1. Clone and setup
git clone https://github.com/javid-p84/jcrawl.git
cd jcrawl
cp .env.example .env

# 2. Update .env with your database URL
# DATABASE_URL=postgres://user:password@localhost:5432/jcrawl?sslmode=disable

# 3. Run
go run main.go

# 4. Visit http://localhost:8080
```

---

## 📖 Overview

A multi-user Go service that monitors online availability and automatically books items based on user preferences. Currently supports restaurant reservations via Google Maps and recreation.gov campground bookings.

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

### Quick Start with Docker (Recommended)

**Prerequisites:**
- Docker
- Docker Compose

**Setup (3 commands):**
```bash
git clone https://github.com/javid-p84/jcrawl.git
cd jcrawl
docker-compose up
```

Server starts on `http://localhost:8080`

Database automatically initializes on first run.

### Local Development (Without Docker)

**Prerequisites:**
- Go 1.21+
- PostgreSQL 12+
- Chrome/Chromium

**Installation:**

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

4. Configure PostgreSQL in `.env`:
```
DATABASE_URL=postgres://user:password@localhost:5432/jcrawl?sslmode=disable
```

5. Build and run:
```bash
go build -o jcrawl
./jcrawl
```

Server starts on `http://localhost:8080`

## How to Use

Once the server is running, open **http://localhost:8080** for a simple web UI — register, log in, and manage preferences/notifications/bookings without touching the API directly. Everything below is the equivalent `curl` walkthrough, useful for scripting or automation.

### 1. Register & Login

```bash
# Register (one-time)
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "secure_password"
  }'

# Login — returns a JWT; save it for every request below
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "secure_password"
  }' | jq -r .token)
```

Every `/api/v1` endpoint except register/login requires `Authorization: Bearer $TOKEN`. Tokens expire after 24 hours — log in again to get a new one.

### 2. Create a Preference

A preference is "watch this booking page, on this schedule, and react this way." Any response is JSON; capture the `id` field (`PREF_ID` below) to manage it later.

**Restaurant example** (Resy, OpenTable, Google Reserve, or any other booking page):
```bash
curl -X POST http://localhost:8080/api/v1/preferences \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "google_link": "https://www.opentable.com/r/some-restaurant",
    "restaurant_name": "Restaurant Name",
    "date_range_from": "2024-01-01",
    "date_range_to": "2024-01-31",
    "day_preference": [5, 6],
    "party_size": 2,
    "guest_name": "Jane Doe",
    "guest_email": "jane@example.com",
    "guest_phone": "+15551234567",
    "auto_book": true
  }'
```

**Recreation.gov example**, with a choice of three modes:

**Mode A — Notify only.** No recreation.gov account needed at all; jcrawl just tells you when a spot opens and you book it yourself. This example asks for a 3-night stay: with `day_preference` set to Fri/Sat/Sun, jcrawl looks specifically for a 3-night block *starting on Friday* (Saturday and Sunday are treated as the rest of that same stay, not separate candidate check-in days).
```bash
curl -X POST http://localhost:8080/api/v1/preferences \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "google_link": "https://www.recreation.gov/camping/campgrounds/232447/",
    "restaurant_name": "Yosemite Valley Campground",
    "date_range_from": "2024-07-01",
    "date_range_to": "2024-07-31",
    "day_preference": [5, 6, 0],
    "consecutive_days": 3,
    "party_size": 4,
    "notify_only": true
  }'
```

`consecutive_days` defaults to 1 (check each preferred day independently, same as before this field existed) and currently only affects recreation.gov checks — restaurant reservations don't have a multi-night concept in jcrawl.

**Mode B — Auto-book with your recreation.gov password.** Set `auto_book: true` and `guest_name`/`guest_email`/`guest_phone` on the preference (as in the restaurant example), then attach credentials. This is the only mode that can currently complete a real booking — jcrawl logs into recreation.gov with these credentials in the same browser session it uses to reserve the site.
```bash
curl -X POST "http://localhost:8080/api/v1/recreation/credentials/password?preference_id=PREF_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "username": "your-email@example.com",
    "password": "your-recreation-gov-password"
  }'
```

**Mode C — Auto-book with an OAuth token.** Stored encrypted and usable for availability checks, but there's no verified way to turn a bearer token into a logged-in browser session, so a booking attempt with only a token configured fails with a clear error rather than pretending to succeed. Use Mode B for actual bookings today.
```bash
curl -X POST "http://localhost:8080/api/v1/recreation/credentials/oauth?preference_id=PREF_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "oauth_token": "your-session-token",
    "oauth_provider": "recreation.gov"
  }'
```

### 3. Manage Your Preferences

```bash
# List everything you're watching
curl http://localhost:8080/api/v1/preferences \
  -H "Authorization: Bearer $TOKEN"

# Update — send only the fields you're changing
curl -X PATCH "http://localhost:8080/api/v1/preferences/PREF_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"party_size": 6}'

# Pause monitoring without deleting it
curl -X PATCH "http://localhost:8080/api/v1/preferences/PREF_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"active": false}'

# Resume it later
curl -X PATCH "http://localhost:8080/api/v1/preferences/PREF_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"active": true}'

# Delete it entirely
curl -X DELETE "http://localhost:8080/api/v1/preferences/PREF_ID" \
  -H "Authorization: Bearer $TOKEN"
```

### 4. Check Notifications

```bash
# Get all notifications (paginated)
curl http://localhost:8080/api/v1/notifications \
  -H "Authorization: Bearer $TOKEN"

# Get unread count
curl http://localhost:8080/api/v1/notifications/unread-count \
  -H "Authorization: Bearer $TOKEN"

# Mark one as read
curl -X POST "http://localhost:8080/api/v1/notifications/mark-as-read?id=NOTIF_ID" \
  -H "Authorization: Bearer $TOKEN"

# Mark all as read
curl -X POST http://localhost:8080/api/v1/notifications/mark-all-as-read \
  -H "Authorization: Bearer $TOKEN"
```

For instant push instead of polling, connect a WebSocket client to `ws://localhost:8080/ws/notifications?token=$TOKEN` — see [NOTIFICATIONS.md](NOTIFICATIONS.md) for a full example.

### 5. View Booking History

```bash
curl http://localhost:8080/api/v1/bookings \
  -H "Authorization: Bearer $TOKEN"
```

A booking only ever shows `status: "booked"` if the automation captured a real confirmation ID from the site — nothing is recorded as successful unless it actually happened.

### 6. Review Check History for a Preference

Every worker pass over a preference — found, not found, or errored — is recorded, not just the ones that triggered a notification or booking. In the web UI, click **"Check history"** on any preference card.

```bash
curl "http://localhost:8080/api/v1/preferences/PREF_ID/checks" \
  -H "Authorization: Bearer $TOKEN"
```

Each entry includes `sites_checked` (how many campsites/slots were examined), `matches_found`, and — if anything matched — `best_match_label`/`best_match_url` pointing at the most likely option (the match with the soonest check-in date).

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Create new account
- `POST /api/v1/auth/login` - Login and get token

### Preferences
- `POST /api/v1/preferences` - Create monitoring preference
- `GET /api/v1/preferences` - List user's preferences
- `PATCH /api/v1/preferences/<id>` - Update a preference (partial; only send fields you're changing). Setting `"active": false` pauses monitoring, `"active": true` resumes it.
- `DELETE /api/v1/preferences/<id>` - Delete a preference
- `GET /api/v1/preferences/<id>/checks` - Check history: every time the worker checked this preference (found or not), how many sites/slots were examined, and the most likely candidate (with a direct link), newest first. Paginated (`?limit=&offset=`).

### Bookings
- `GET /api/v1/bookings` - List user's booking history

### Notifications
- `GET /api/v1/notifications` - Get user's notifications (paginated)
- `GET /api/v1/notifications/unread-count` - Get count of unread notifications
- `POST /api/v1/notifications/mark-as-read?id=<notif-id>` - Mark notification as read
- `POST /api/v1/notifications/mark-all-as-read` - Mark all notifications as read

### Recreation.gov Authentication (2 Options)
**Option 1: Username/Password (Encrypted)**
- `POST /api/v1/recreation/credentials/password?preference_id=<id>` - Store recreation.gov email/password

**Option 2: OAuth Token**
- `POST /api/v1/recreation/credentials/oauth?preference_id=<id>` - Store recreation.gov OAuth token

### Health
- `GET /health` - Service health check

See [How to Use](#how-to-use) above for a complete walkthrough with real request bodies.

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

## Docker

jcrawl includes full Docker support for easy deployment.

### Docker Compose Setup

**Start everything:**
```bash
docker-compose up
```

**In the background:**
```bash
docker-compose up -d
```

**Stop everything:**
```bash
docker-compose down
```

**View logs:**
```bash
docker-compose logs -f jcrawl
docker-compose logs -f db
```

### What's Included

- **jcrawl service** - Go application with Chrome/Chromium
- **PostgreSQL database** - Automatic initialization
- **Health checks** - Automatic restart on failure
- **Volume persistence** - Database data persists between restarts
- **Networking** - Internal Docker network for service communication

### Docker Production Deployment

For production, update `.env` with:
```
ENCRYPTION_KEY=<generate-secure-32-byte-key>
JWT_SECRET=<generate-secure-random-value>
POSTGRES_PASSWORD=<strong-password>
SERVER_ENV=production
LOG_LEVEL=warn
```

Then:
```bash
docker-compose up -d
```

### Rebuild After Code Changes

```bash
docker-compose down
docker-compose build
docker-compose up
```

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

## In-App Notifications

All events are tracked as in-app notifications accessible via the API. Notification types:

| Type | When | Example |
|------|------|---------|
| **availability_found** | Slot becomes available matching preferences | "✨ Michelin Star Restaurant has availability on Jan 20, 2024 at 7:30 PM" |
| **booking_success** | Reservation successfully completed | "🎊 Your reservation at Restaurant is confirmed. Confirmation: RESY-12345" |
| **booking_failed** | Booking attempt fails | "⚠️ Could not complete booking. Reason: Form field validation error" |
| **check_complete** | Availability check finishes (optional) | "📋 Check completed for Restaurant. Found 2 available slot(s)." |
| **error** | An error occurs during monitoring | "❌ Error checking Restaurant: Browser timeout" |

**Notification Features:**
- ✅ Persistent storage in database
- ✅ Mark as read/unread
- ✅ Pagination support
- ✅ Unread count tracking
- ✅ Rich data (restaurant, date, time, confirmation ID)

**Example Notification:**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "preference_id": "uuid",
  "type": "booking_success",
  "title": "🎊 Booking Confirmed!",
  "message": "Your reservation at Michelin Star Restaurant for Jan 20, 2024 at 7:30 PM is confirmed. Confirmation: RESY-123456",
  "read": false,
  "data": {
    "restaurant": "Michelin Star Restaurant",
    "date": "2024-01-20",
    "time": "7:30 PM",
    "confirmation_id": "RESY-123456"
  },
  "created_at": "2024-01-17T14:30:00Z"
}
```

## Booking Platforms Supported

| Platform | Type | Format | Status |
|----------|------|--------|--------|
| **Resy** | Restaurant | data-time attributes, form inputs | ✅ Implemented |
| **OpenTable** | Restaurant | Button-based time slots | ✅ Implemented |
| **Google Reserve** | Restaurant | Dialog-based booking | ✅ Implemented |
| **Recreation.gov** | Camping/Outdoors | API + browser automation | ✅ Implemented |
| **Generic** | Any | Text-based times + common inputs | ✅ Fallback |

### Recreation.gov Features

Recreation.gov support includes:
- ✅ Campground availability checking via API
- ✅ Day-use area reservations
- ✅ Facility ID extraction from URLs
- ✅ Multi-date availability checks
- ✅ Day-of-week filtering
- ✅ Automatic reservation completion

**Supported recreation.gov URLs:**
- `https://www.recreation.gov/camping/campgrounds/123456/`
- `https://www.recreation.gov/camping/campsites/123456/`
- `https://www.recreation.gov/api/camps/availability/campgrounds/123456/month/...`

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

3. User gets notified:
   - Instantly via WebSocket, plus email/SMS if configured (see NOTIFICATIONS.md)
   - Booking record saved in the database
   - Preference auto-deactivated (for the auto-book case)
```

## Booking Modes

jcrawl supports **three usage modes** per preference — see [How to Use](#how-to-use) for the actual request bodies. Summary:

| Mode | Credentials needed | Books automatically? |
|---|---|---|
| **Notify only** (`notify_only: true`) | None | No — you book manually when notified |
| **Auto-book, password** | recreation.gov username/password | Yes — the only mode that can complete a real booking today |
| **Auto-book, OAuth token** | recreation.gov OAuth token | No — refused with an explanatory error; the token can check availability but can't drive an authenticated browser session |

## Recreation.gov Authentication Options

See [How to Use](#how-to-use) for the request bodies to store each credential type. Both are encrypted (AES-256-GCM) before being saved and are never returned by the API.

**Username/password** — simple, and the only option that currently completes bookings. Downside: you're storing a password (encrypted) and have to update it if it changes.

**OAuth token** — no password stored, and can be revoked independently of your account password. Get it from an existing logged-in browser session (DevTools → Application → Cookies, or your session/auth cookie) and paste it in. Downside: tokens expire (recreation.gov's typically last a few weeks) and, as above, can't currently complete an actual booking — only availability checks.

## Security & Encryption

jcrawl uses **AES-256-GCM encryption** for storing sensitive credentials:

### Credential Storage
- **Option 1: Passwords** - Encrypted using AES-256-GCM before storage
- **Option 2: OAuth Tokens** - Encrypted using AES-256-GCM before storage
- **Encryption key** - 32-byte key from `ENCRYPTION_KEY` environment variable
- **Storage** - Base64-encoded ciphertext in PostgreSQL
- **Decryption** - Only when needed for login/booking
- **Token validation** - Optional automatic expiry checking

### Security Features
✅ **Never logged** - Passwords never appear in logs  
✅ **Encrypted at rest** - Stored as encrypted ciphertext in database  
✅ **Unique nonce** - Each encryption uses a random nonce (IV)  
✅ **Authenticated encryption** - GCM mode prevents tampering  
✅ **Separate keys** - Encryption key separate from app secrets  

### Setting Up Encryption Key

The `ENCRYPTION_KEY` is a passphrase of any length; the AES-256 key is derived from it via SHA-256. Generate a strong one:
```bash
openssl rand -hex 32
```

Set in `.env`:
```
ENCRYPTION_KEY=your-generated-passphrase-here
```

**Warning:** changing the passphrase later makes previously stored credentials unreadable.

### How Recreation.gov Login Works

**Storage Phase:**
```
User Input: password "MyPassword123"
         ↓
Crypto.Encrypt(password)
         ↓
Encrypted: "FsK8x9L2Np3qR5sT7uV9w1xY3zA5bC7d..." (base64)
         ↓
Store in DB
```

**Usage Phase (During Booking):**
```
Read encrypted password from DB
         ↓
Crypto.Decrypt(encrypted)
         ↓
Plaintext: "MyPassword123"
         ↓
Use for login/booking
         ↓
Memory cleared
```

### API Usage

**Store recreation.gov credentials:**
```bash
curl -X POST "http://localhost:8080/api/v1/recreation/credentials/password?preference_id=UUID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "username": "your-email@example.com",
    "password": "your-recreation-gov-password"
  }'
```

**Response:**
```json
{
  "status": "ok",
  "message": "Credentials stored (encrypted). Please ensure your recreation.gov username and password are correct."
}
```

## Troubleshooting

### Docker Issues

**Port already in use:**
```bash
# Use different port
# Edit docker-compose.yml, change "8080:8080" to "8081:8080"
docker-compose up
```

**Database connection failed:**
```bash
# Check if services are running
docker-compose ps

# View database logs
docker-compose logs db

# Restart everything
docker-compose down
docker-compose up
```

**Browser automation not working:**
```bash
# Chrome/Chromium must be in container
# Verify it's installed in Dockerfile
docker-compose logs jcrawl | grep -i chrome
```

### Local Development Issues

**Port 8080 in use:**
```bash
# Find what's using the port
lsof -i :8080

# Or use different port
SERVER_PORT=8081 go run main.go
```

**Database connection failed:**
```bash
# Verify PostgreSQL is running
psql -U jcrawl -d jcrawl

# Check .env DATABASE_URL format
cat .env | grep DATABASE_URL
```

**Chrome/Chromium not found:**
```bash
# Linux (Ubuntu/Debian)
sudo apt-get install chromium-browser

# macOS
brew install chromium

# macOS (with system Chrome)
# Just use /Applications/Google\ Chrome.app/...
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

## Support

- 📖 Read [DEPLOYMENT.md](DEPLOYMENT.md) for production setup
- 🐛 Report issues on [GitHub Issues](https://github.com/javid-p84/jcrawl/issues)
- 💬 Discuss ideas in [GitHub Discussions](https://github.com/javid-p84/jcrawl/discussions)
