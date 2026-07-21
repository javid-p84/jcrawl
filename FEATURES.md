# jcrawl Features

A reference of everything jcrawl currently does. For setup, see [README.md](README.md); for notification setup, see [NOTIFICATIONS.md](NOTIFICATIONS.md); for production deployment, see [DEPLOYMENT.md](DEPLOYMENT.md).

## Core Concept

jcrawl monitors a booking page (restaurant reservation, campground, etc.) on a schedule and, depending on how a preference is configured, either notifies the user when a slot opens up or books it automatically.

## Accounts & Authentication

- Email/password registration (`POST /api/v1/auth/register`), password hashed with bcrypt
- Login (`POST /api/v1/auth/login`) returns a signed JWT (HS256, 24h expiry)
- Every `/api/v1` endpoint except register/login requires `Authorization: Bearer <token>`
- All data (preferences, bookings, notifications, credentials) is scoped to the authenticated user — one user cannot read or modify another's records
- WebSocket connections authenticate the same JWT via a `?token=` query parameter (browsers can't set custom headers when opening a socket)

## Monitoring Preferences

A preference describes what to watch for and how to react. Fields:

- `google_link` — URL of the restaurant or facility to monitor
- `date_range_from` / `date_range_to` — window to check within
- `day_preference` — which days of the week count (e.g. Friday/Saturday only)
- `consecutive_days` — how many nights in a row are needed, starting on the first day of a `day_preference` run (recreation.gov only; see below)
- `party_size`
- `guest_name` / `guest_email` / `guest_phone` / `special_notes` — used to fill the booking form
- `auto_book` — book automatically when a match is found
- `notify_only` — just notify, never book (overrides `auto_book`)
- `active` — whether the preference is currently being checked

**Full CRUD:**
- `POST /api/v1/preferences` — create
- `GET /api/v1/preferences` — list your preferences
- `PATCH /api/v1/preferences/{id}` — partial update; also used to pause (`"active": false`) or resume (`"active": true`) a preference
- `DELETE /api/v1/preferences/{id}` — remove

### Consecutive-night stays (camping)

`day_preference` identifies which day a stay may *start* on — for a run of consecutive preferred weekdays (e.g. Friday, Saturday, Sunday), only the first day of that run is used as a candidate check-in date. Saturday and Sunday are covered by the stay length rather than treated as separate starting points. `consecutive_days` (default 1) is how many nights in a row, starting on that day, must be available on the *same* campsite.

Example: `day_preference: [5, 6, 0]` (Fri, Sat, Sun) with `consecutive_days: 3` looks only for 3-night blocks starting on a Friday — never a 3-night block starting Saturday or Sunday. This works for any run, including ones that wrap the week boundary (e.g. Saturday/Sunday), and for non-contiguous preferences (each preferred day is evaluated as its own independent start).

This only affects recreation.gov checks; restaurant reservations are single time slots with no multi-night concept in jcrawl.

## Background Monitoring

- A worker checks all active preferences on a fixed interval (default every 5 minutes, configurable via `WORKER_CHECK_INTERVAL_MINUTES`)
- Up to 5 preferences are checked concurrently
- Each check updates `last_checked_at` on the preference regardless of outcome
- Availability is matched against the preference's date range and day-of-week filter before anything else happens

### Check history

Every worker pass over a preference is recorded — found, not found, or errored — not just the ones that produced a notification or booking. Each record has:

- Success/failure (and the error message, if it failed)
- `sites_checked` — how many campsites/slots were examined (for recreation.gov, distinct sites seen across the months fetched; for restaurants, distinct dates queried)
- `matches_found`
- The most likely candidate among the matches (soonest check-in date), with a direct link — for recreation.gov, a link straight to that specific campsite's page

View it in the web UI via the **"Check history"** button on each preference card, or `GET /api/v1/preferences/{id}/checks` (paginated, scoped to the requesting user).

## Supported Booking Platforms

Detected automatically from the `google_link` URL:

| Platform | Availability check | Auto-booking |
|---|---|---|
| **Recreation.gov campgrounds** (campsites, day-use areas) | Public JSON API (no login required) | Browser automation, requires stored username/password |
| **Recreation.gov permits** (wilderness/overnight permits, e.g. `recreation.gov/permits/{id}/...`) | Public JSON API (no login required) | Not supported — refused with an explanation; use `notify_only` |
| **Resy** | Browser scrape | Browser automation |
| **OpenTable** | Browser scrape | Browser automation |
| **Google Reserve** | Browser scrape | Browser automation |
| Any other booking page | Generic pattern-matching scrape | Generic browser automation (best-effort form fill) |

Browser-based checks and bookings run through headless Chrome (chromedp).

### Recreation.gov permits

Permits (`/permits/{id}/...` URLs) are a separate recreation.gov subsystem from campgrounds — different API, different data shape — detected and routed independently even though both are recreation.gov URLs:

- Availability is a **quota headcount per entry date and division** (trailhead/zone), not a boolean per-site/per-night like campsites. A date matches if a division's remaining quota covers the preference's `party_size` (minimum 1).
- `consecutive_days` doesn't apply to permits — a permit entry-date quota isn't a multi-night stay tied to one site, so each preferred day is checked independently.
- The matched "site" is the division name (e.g. "Mt. Whitney Trail (Overnight)"), and the link points back to that specific date's registration page.
- **Auto-booking is explicitly refused, not attempted.** Permit registration typically involves waivers, fees, and trip itineraries — meaningfully more complex than a restaurant or campsite reservation, and nothing about that flow has been observed or automated here. A permit preference with `auto_book: true` will fail loudly with an explanatory error rather than run a guessed, unverified flow against a real transaction. Use `notify_only` for permits and complete registration yourself.

The endpoints used here (`/api/permitinyo/{id}/availabilityv2` and `/api/permitcontent/{id}`) were captured from real browser network traffic against a live permit page, not guessed from the campground API's shape — they're a genuinely different API. One quirk worth knowing: the availability endpoint only accepts date ranges that are exactly a calendar month (`start_date`/`end_date` = first/last day of the same month), so checks fetch one full month at a time and filter down to the preference's actual requested range, mirroring how the campground scraper already handles months.

## Recreation.gov Authentication

Two ways to let jcrawl access your recreation.gov account, both AES-256-GCM encrypted at rest:

- **Username/password** (`POST /api/v1/recreation/credentials/password`) — the only method that can currently complete an actual booking. The worker logs in with these credentials in the same browser session used to make the reservation.
- **OAuth token** (`POST /api/v1/recreation/credentials/oauth`) — stored encrypted and usable for authenticated API calls (e.g. availability checks), but cannot currently drive an authenticated *browser* session, so auto-booking with only a token configured is refused with a clear error rather than attempted and silently failing.

Neither credential type is ever returned by the API once stored.

## Three Ways to Use a Preference

1. **Notify-only** (`notify_only: true`) — no credentials needed at all. jcrawl watches and tells you the moment something opens up; you book manually.
2. **Auto-book with password** — jcrawl logs in and completes the reservation for you.
3. **Auto-book with OAuth token** — currently limited to availability checking; see above.

A preference is automatically deactivated after a successful auto-booking so it stops re-checking (and re-booking) the same slot.

## Booking Safety

- A booking is only ever recorded as successful if the automation actually captures a real confirmation ID from the booking site — nothing is fabricated. A failed or incomplete flow is recorded and reported as a failure, never as a success.
- Every booking attempt (success or failure) is written to `booking_history` with status, confirmation ID (if any), and notes.
- `GET /api/v1/bookings` returns your full booking history.

## Notifications

Every meaningful event generates an in-app notification, stored permanently and delivered through multiple channels at once:

- **WebSocket** — instant, real-time push to any connected client (`/ws/notifications?token=<jwt>`)
- **Email** — via SMTP (Gmail or any provider), HTML formatted, with a link back to jcrawl
- **SMS** — via Twilio (channel implemented; actual send call is a stub pending Twilio wiring)

Each channel is optional and auto-registers only if its environment variables are configured. Delivery retries up to 3 times per channel with exponential backoff, and channels are sent to concurrently so a slow email server doesn't delay the WebSocket push.

**Notification types:**
- Availability found
- Booking succeeded
- Booking failed (including "credentials missing" for recreation.gov auto-book)
- Check complete
- Error

**API:**
- `GET /api/v1/notifications` — paginated list
- `GET /api/v1/notifications/unread-count`
- `POST /api/v1/notifications/mark-as-read?id=<id>`
- `POST /api/v1/notifications/mark-all-as-read`

## Security

- Passwords hashed with bcrypt (never stored or logged in plaintext)
- Recreation.gov credentials (password and OAuth token) encrypted with AES-256-GCM before being written to the database; the encryption key is derived via SHA-256 from the `ENCRYPTION_KEY` passphrase
- JWT-based authentication on every user-scoped endpoint, with per-request ownership checks at the database layer (not just the API layer)
- No credential or token value is ever included in an API response or a log line

## Web UI

A single-page UI is served directly by the app at `/` — no separate build step or frontend server, and no extra deployment step since it's embedded in the compiled Go binary.

- Register / log in (JWT stored in the browser, sent as `Authorization: Bearer`)
- Create, edit, pause/resume, and delete preferences, including the day-of-week picker and notify-only vs. auto-book toggle
- A one-click form on each recreation.gov preference to store the account password used for auto-booking
- Notifications list with unread badge, mark-as-read / mark-all-as-read, and instant toast popups pushed live over the WebSocket connection
- Booking history with status and confirmation ID

## Deployment

- Single `Dockerfile` (multi-stage build, includes headless Chrome) and `docker-compose.yml` (app + PostgreSQL) — `docker-compose up` is the entire setup
- PostgreSQL schema is created automatically on startup if it doesn't exist
- Health check endpoint: `GET /health`
- See [DEPLOYMENT.md](DEPLOYMENT.md) for production hardening (secrets, reverse proxy, backups, scaling)

## Not Yet Built

Tracked here so it's clear what's aspirational vs. real:

- Actual Twilio API call in the SMS channel (currently logs what it would send)
- OAuth-token-based auto-booking for recreation.gov
- Booking cancellation / cancellation monitoring
- Support for hotels, flights, tickets, and other booking categories
- Refresh tokens (JWTs simply expire after 24h and require re-login)
