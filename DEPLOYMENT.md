# jcrawl Deployment Guide

## Local Development

### With Docker Compose (Recommended)

```bash
docker-compose up
```

App available at: `http://localhost:8080`

Database: `localhost:5432` (user: `jcrawl`, password: `jcrawl_secure_password_change_me`)

### Without Docker

1. Install PostgreSQL 12+
2. Create database:
   ```sql
   CREATE DATABASE jcrawl;
   ```

3. Create user:
   ```sql
   CREATE USER jcrawl WITH PASSWORD 'password';
   GRANT ALL PRIVILEGES ON DATABASE jcrawl TO jcrawl;
   ```

4. Install Go 1.21+
5. Install Chrome/Chromium
6. Set `.env`:
   ```
   DATABASE_URL=postgres://jcrawl:password@localhost:5432/jcrawl?sslmode=disable
   ENCRYPTION_KEY=<generate-32-byte-key>
   JWT_SECRET=<random-value>
   ```

7. Run:
   ```bash
   go run main.go
   ```

## Production Deployment

### Prerequisites

- Docker & Docker Compose installed
- Server with at least:
  - 2 CPU cores
  - 2GB RAM
  - 10GB disk space
- Domain name (for HTTPS/reverse proxy)

### Step 1: Generate Secure Keys

```bash
# Encryption key (32 bytes)
openssl rand -hex 16

# JWT secret
openssl rand -base64 32

# Database password
openssl rand -base64 16
```

### Step 2: Prepare .env for Production

```bash
cp .env.example .env.production
```

Update with:
```env
DATABASE_URL=postgres://jcrawl:STRONG_PASSWORD@db:5432/jcrawl?sslmode=disable
SERVER_ENV=production
SERVER_PORT=8080
ENCRYPTION_KEY=YOUR_32_BYTE_KEY
JWT_SECRET=YOUR_RANDOM_JWT_SECRET
LOG_LEVEL=warn
WORKER_CHECK_INTERVAL_MINUTES=5
```

### Step 3: Update docker-compose.yml

Change passwords in `docker-compose.yml`:
```yaml
services:
  db:
    environment:
      POSTGRES_PASSWORD: STRONG_PASSWORD
```

### Step 4: Deploy

```bash
# Build images
docker-compose build

# Start services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f jcrawl
```

### Step 5: Set Up Reverse Proxy (Nginx)

```nginx
server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Monitoring

### Health Check

```bash
curl http://localhost:8080/health
```

### Database Connection

```bash
docker-compose exec db psql -U jcrawl -d jcrawl -c "SELECT COUNT(*) FROM users;"
```

### View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f jcrawl
docker-compose logs -f db

# Recent logs (last 100 lines)
docker-compose logs --tail=100 jcrawl
```

## Maintenance

### Backup Database

```bash
docker-compose exec db pg_dump -U jcrawl jcrawl > jcrawl_backup.sql
```

### Restore Database

```bash
docker-compose exec -T db psql -U jcrawl jcrawl < jcrawl_backup.sql
```

### Update Application

```bash
# Pull latest code
git pull origin main

# Rebuild and restart
docker-compose down
docker-compose build
docker-compose up -d
```

### Clean Up

```bash
# Remove stopped containers
docker-compose rm

# Remove unused volumes
docker volume prune

# Remove unused images
docker image prune
```

## Troubleshooting

### Database Connection Failed

```bash
# Check if database is running
docker-compose ps db

# View database logs
docker-compose logs db

# Try to connect directly
docker-compose exec db psql -U jcrawl -d jcrawl -c "\dt"
```

### Application Won't Start

```bash
# View full logs
docker-compose logs jcrawl

# Rebuild containers
docker-compose build --no-cache

# Restart
docker-compose restart
```

### Port Already in Use

Change in `docker-compose.yml`:
```yaml
services:
  jcrawl:
    ports:
      - "8081:8080"  # Use 8081 instead of 8080
```

## Performance Tuning

### Increase Worker Concurrency

Edit `pkg/worker/worker.go`:
```go
semaphore := make(chan struct{}, 10)  // Increase from 5 to 10
```

### Database Connection Pool

In `pkg/db/db.go`:
```go
db.SetMaxOpenConns(50)   // Increase connections
db.SetMaxIdleConns(10)   // Increase idle
```

### Check Interval

In `.env`:
```
WORKER_CHECK_INTERVAL_MINUTES=3  # Check every 3 minutes instead of 5
```

## Scaling

For multiple instances:
- Use PostgreSQL on separate server
- Use load balancer (HAProxy, Nginx)
- Scale jcrawl containers horizontally
- Consider managed database service (RDS, etc.)
