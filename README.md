# jcrawl

A Go service that monitors online availability and automatically books items based on your preferences. Supports a variety of booking types (restaurant reservations, appointments, tickets, etc.).

## Features

- Monitor restaurant availability from Google Maps
- Check every 5 minutes (configurable)
- Auto-book when availability matches your preferences
- Filter by date range and day of week
- Support for party size preferences
- Graceful shutdown and error handling

## Getting Started

### Prerequisites

- Go 1.21 or later

### Build

```bash
go build -o jcrawl
```

### Configuration

Copy `.env.example` to `.env` and update with your preferences:

```bash
cp .env.example .env
```

Edit `.env` with:
- `RESTAURANT_GOOGLE_LINK`: Google Maps link to the restaurant
- `RESTAURANT_PARTY_SIZE`: Number of people in your party
- `DATE_RANGE_FROM` and `DATE_RANGE_TO`: When you want to book (YYYY-MM-DD format)
- `DAY_PREFERENCE`: Preferred days (0=Sunday, 1=Monday, ..., 6=Saturday)
- `CHECK_INTERVAL_MINUTES`: How often to check availability (default: 5)
- `AUTO_BOOK`: Auto-book when availability is found (default: true)

### Run

```bash
export $(cat .env | xargs)
./jcrawl
```

Or with Go:

```bash
export $(cat .env | xargs)
go run main.go
```

## Development

```bash
go mod tidy
go run main.go
```

## Architecture

- `pkg/models/` - Data structures for preferences and availability
- `pkg/restaurant/` - Restaurant availability checking
- `pkg/scheduler/` - Scheduling and monitoring logic
- `pkg/config/` - Configuration management
