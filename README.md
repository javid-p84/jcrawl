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

### 1. Register & Login

```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "secure_password"
  }'

# Login — returns a JWT; save it for all subsequent requests
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "secure_password"
  }' | jq -r .token)
```

All `/api/v1` endpoints (except register/login) require the token via `Authorization: Bearer $TOKEN`. Tokens expire after 24 hours.

### 2. Create a Preference (Example: Recreation.gov)

**Option A: Notifications Only (No Login Required)**
```bash
curl -X POST http://localhost:8080/api/v1/preferences \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "google_link": "https://www.recreation.gov/camping/campgrounds/232447/",
    "restaurant_name": "Yosemite Valley Campground",
    "date_range_from": "2024-07-01",
    "date_range_to": "2024-07-31",
    "day_preference": [5, 6],
    "party_size": 4,
    "notify_only": true,
    "auto_book": false
  }'
```

**Option B: Auto-Book with Password**
```bash
curl -X POST http://localhost:8080/api/v1/recreation/credentials/password?preference_id=PREF_UUID \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "username": "your-email@example.com",
    "password": "your-recreation-gov-password"
  }'
```

**Option C: Auto-Book with OAuth Token**
```bash
curl -X POST http://localhost:8080/api/v1/recreation/credentials/oauth?preference_id=PREF_UUID \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "oauth_token": "your-session-token",
    "oauth_provider": "recreation.gov"
  }'
```

### 3. Check Notifications

```bash
# Get all notifications
curl http://localhost:8080/api/v1/notifications \
  -H "Authorization: Bearer $TOKEN"

# Get unread count
curl http://localhost:8080/api/v1/notifications/unread-count \
  -H "Authorization: Bearer $TOKEN"

# Mark as read
curl -X POST http://localhost:8080/api/v1/notifications/mark-as-read?id=NOTIF_UUID \
  -H "Authorization: Bearer $TOKEN"
```

### 4. View Bookings

```bash
curl http://localhost:8080/api/v1/bookings \
  -H "Authorization: Bearer $TOKEN"
```

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Create new account
- `POST /api/v1/auth/login` - Login and get token

### Preferences
- `POST /api/v1/preferences` - Create monitoring preference
- `GET /api/v1/preferences` - List user's preferences

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

## Example Usage

```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secure123"}'

# Add restaurant preference
curl -X POST http://localhost:8080/api/v1/preferences \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
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
  -H "Authorization: Bearer $TOKEN"
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

3. User gets confirmation:
   - Email sent (future feature)
   - Booking record in database
   - Preference auto-deactivated
```

## Booking Modes

jcrawl supports **THREE flexible usage modes**:

### 1️⃣ Notifications Only (No Login Required)
- Check availability without storing any credentials
- Get in-app notifications when campsites become available
- Manually book through recreation.gov website
- Perfect for casual checking

**Setup:**
```bash
curl -X POST http://localhost:8080/api/v1/preferences \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "google_link": "https://www.recreation.gov/camping/campgrounds/123456/",
    "restaurant_name": "Yosemite Valley Campground",
    "date_range_from": "2024-07-01",
    "date_range_to": "2024-07-31",
    "day_preference": [5, 6],
    "party_size": 4,
    "notify_only": true,
    "auto_book": false
  }'
```

**Result:**
- ✅ No credentials needed
- ✅ Get notifications when availability found
- ✅ Browse/click to book manually
- ⏭️ Simple, low-risk

### 2️⃣ Auto-Book with Option 1: Username/Password
- Store email/password securely
- Automatically book when availability is found
- Fastest way to secure a campsite

### 3️⃣ Auto-Book with Option 2: OAuth Token
- Store recreation.gov OAuth token
- Automatically book with token authentication
- No password stored

## Recreation.gov Authentication Options

jcrawl supports **TWO secure authentication methods** (when using auto-book):

### Option 1: Username/Password (Encrypted)
Store your recreation.gov email and password securely:
```bash
curl -X POST http://localhost:8080/api/v1/recreation/credentials/password?preference_id=UUID \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "username": "your-email@example.com",
    "password": "your-recreation-gov-password"
  }'
```

**Pros:**
- ✅ Simple setup
- ✅ AES-256-GCM encrypted
- ✅ Works with all recreation.gov features

**Cons:**
- ⚠️ Requires password storage
- ⚠️ Password changes require update

### Option 2: OAuth Token
Use a recreation.gov session token (copy from browser):
```bash
curl -X POST http://localhost:8080/api/v1/recreation/credentials/oauth?preference_id=UUID \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "oauth_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "oauth_provider": "recreation.gov",
    "oauth_refresh": "optional-refresh-token"
  }'
```

**How to get your OAuth token:**
1. Login to recreation.gov
2. Open browser DevTools (F12)
3. Go to Application → Cookies
4. Find `JSESSIONID` or auth token
5. Copy and paste into jcrawl

**Pros:**
- ✅ No password stored
- ✅ Works with Google/Facebook OAuth
- ✅ Can use existing logged-in session
- ✅ Tokens can be revoked independently

**Cons:**
- ⚠️ Tokens expire (typically 24-30 days)
- ⚠️ Need to refresh periodically
- ⚠️ Slight manual setup required

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
curl -X POST http://localhost:8080/api/v1/recreation/credentials?preference_id=UUID \
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
  "message": "Credentials updated. Please ensure your recreation.gov username and password are correct."
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
