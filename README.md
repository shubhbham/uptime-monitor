# Uptime Monitor API

A robust and scalable website/endpoint monitoring system built with Go and Fiber, similar to UptimeRobot. Monitor your websites, APIs, and services with configurable intervals, receive downtime alerts, and track comprehensive uptime statistics.

## Features

- 🚀 **Real-time Monitoring**: Monitor HTTP/HTTPS endpoints with customizable intervals (30-3600 seconds)
- 📊 **Detailed Statistics**: Track uptime percentage, response times, and historical data
- 🔔 **Incident Management**: Automatic incident creation and resolution tracking
- ⚡ **High Performance**: Built with Fiber framework for blazing-fast performance
- 🗄️ **PostgreSQL Database**: Reliable data persistence with Supabase
- 🔄 **Concurrent Checks**: Efficient worker pool for parallel monitoring
- 📈 **Rich API**: RESTful API for complete monitor management
- 🛡️ **Error Handling**: Comprehensive error handling and logging

## Architecture

```
┌─────────────────┐
│   API Layer     │  Fiber HTTP Server
│   (Handlers)    │  Routes & Validation
└────────┬────────┘
         │
┌────────▼────────┐
│  Service Layer  │  Business Logic
│   (Services)    │  Orchestration
└────────┬────────┘
         │
┌────────▼────────┐
│Repository Layer │  Data Access
│  (Repositories) │  Database Queries
└────────┬────────┘
         │
┌────────▼────────┐
│   PostgreSQL    │  Supabase Database
│    Database     │  (monitors, checks, incidents)
└─────────────────┘

┌─────────────────┐
│   Scheduler     │  Cron-based Job Scheduler
│   (Background)  │  Periodic Monitor Checks
└────────┬────────┘
         │
┌────────▼────────┐
│  HTTP Checker   │  HTTP Client Worker
│   (Workers)     │  Endpoint Verification
└─────────────────┘
```

### Component Breakdown

1. **API Layer** (`internal/modules/*/handler.go`)
   - Handles HTTP requests/responses
   - Request validation
   - Response formatting

2. **Service Layer** (`internal/modules/*/service.go`)
   - Business logic implementation
   - Coordinates between repositories
   - Data transformation

3. **Repository Layer** (`internal/modules/*/repository.go`)
   - Database operations (CRUD)
   - Query execution
   - Data persistence

4. **Scheduler** (`internal/scheduler/`)
   - Cron-based job scheduling
   - Monitor job management
   - Concurrent execution

5. **Workers** (`internal/worker/`)
   - HTTP health checks
   - Timeout management
   - Response validation

6. **Domain Models** (`internal/domain/`)
   - Shared data structures
   - Prevents circular dependencies
   - Type definitions

## Database Schema

```
monitors (Configuration)
├── id (UUID)
├── name (text)
├── url (text)
├── method (text)
├── expected_status (int)
├── interval_seconds (int)
├── timeout_seconds (int)
└── is_active (boolean)

monitor_checks (Historical Data)
├── id (bigserial)
├── monitor_id (UUID) → monitors
├── status_code (int)
├── response_time_ms (int)
├── is_up (boolean)
├── error_message (text)
└── checked_at (timestamptz)

incidents (Downtime Events)
├── id (bigserial)
├── monitor_id (UUID) → monitors
├── started_at (timestamptz)
├── resolved_at (timestamptz)
├── cause (text)
└── is_resolved (boolean)

monitor_stats (Cached Metrics)
├── monitor_id (UUID) → monitors
├── uptime_percentage (numeric)
├── avg_response_time_ms (int)
├── last_checked_at (timestamptz)
└── last_status (boolean)
```

## Installation

### Prerequisites

- Go 1.24 or higher
- PostgreSQL (Supabase recommended)
- Make (optional, for Makefile commands)

### Setup

1. **Clone the repository**
```bash
git clone <repository-url>
cd uptime-monitor
```

2. **Install dependencies**
```bash
go mod download
# or
make deps
```

3. **Configure environment variables**

Create a `.env` file in the root directory:

```properties
# Server Configuration
PORT=5000
READ_TIMEOUT=10
WRITE_TIMEOUT=10
ENVIRONMENT=development

# Database Configuration (Supabase)
DATABASE_URL=postgresql://postgres.[projectID]:[Password]@aws-1-ap-south-1.pooler.supabase.com:5432/postgres
DB_SSL_MODE=require
DB_MAX_CONNS=25
DB_MIN_CONNS=5

# Monitor Configuration
WORKER_POOL_SIZE=10
MAX_RETRIES=3
DEFAULT_TIMEOUT=10
STATS_UPDATE_PERIOD=60
```

4. **Initialize database**

Run the schema file on your Supabase instance:
```bash
psql $DATABASE_URL -f sql/schema.sql
```

5. **Build and run**
```bash
# Build
make build

# Run
make run
# or
./bin/server
```

## API Endpoints

### Health Check
```
GET /api/v1/health
```

### Monitors

**Create Monitor**
```http
POST /api/v1/monitors
Content-Type: application/json

{
  "name": "My Website",
  "url": "https://example.com",
  "method": "GET",
  "expected_status": 200,
  "interval_seconds": 60,
  "timeout_seconds": 10,
  "notify": true
}
```

**List All Monitors**
```http
GET /api/v1/monitors
```

**Get Monitor**
```http
GET /api/v1/monitors/:id
```

**Update Monitor**
```http
PUT /api/v1/monitors/:id
Content-Type: application/json

{
  "name": "Updated Name",
  "url": "https://example.com",
  "method": "GET",
  "expected_status": 200,
  "interval_seconds": 60,
  "timeout_seconds": 10,
  "notify": true,
  "is_active": false
}
```

**Delete Monitor**
```http
DELETE /api/v1/monitors/:id
```

**Get Monitor Checks**
```http
GET /api/v1/monitors/:id/checks?limit=50
```

### Incidents

**List All Incidents**
```http
GET /api/v1/incidents?limit=50
```

**Get Recent Incidents**
```http
GET /api/v1/incidents/recent?hours=24
```

**Get Monitor Incidents**
```http
GET /api/v1/monitors/:monitor_id/incidents?limit=50
```

### Statistics

**Get Monitor Stats**
```http
GET /api/v1/stats/:monitor_id
```

**Get Uptime Stats**
```http
GET /api/v1/stats/:monitor_id/uptime
```

Response:
```json
{
  "last_24_hours": 99.87,
  "last_7_days": 99.95,
  "last_30_days": 99.99
}
```

```
### User

**Get User Stats**
```http
GET /api/v1/auth/stats
```

## Configuration

### Monitor Settings

- **interval_seconds**: 30-3600 (How often to check)
- **timeout_seconds**: 1-120 (Request timeout)
- **expected_status**: 100-599 (Expected HTTP status)
- **method**: GET, POST, PUT, DELETE, PATCH, HEAD

### Server Settings

- **PORT**: Server port (default: 5000)
- **READ_TIMEOUT**: Read timeout in seconds
- **WRITE_TIMEOUT**: Write timeout in seconds
- **ENVIRONMENT**: development/production

### Database Settings

- **DB_MAX_CONNS**: Maximum connections (default: 25)
- **DB_MIN_CONNS**: Minimum connections (default: 5)
- **DB_SSL_MODE**: SSL mode (require/disable)

### Email Configuration (Brevo)

- **BREVO_API_KEY**: Your Brevo API Key
- **BREVO_SANDBOX_MODE**: true/false (If true, logs email instead of sending)
- **BREVO_API_BASE_URL**: API Base URL (default: https://api.brevo.com)
- **ALERT_FROM_EMAIL**: Sender email address
- **ALERT_FROM_NAME**: Sender name
- **ALERT_TAG_INCIDENT**: Tag for incident emails
- **ALERT_TAG_SERVICE**: Tag for service identification
- **ALERTS_ENABLED**: true/false (Master switch for alerts)
- **ALERT_REPLY_TO_EMAIL**: Reply-to email address
- **ALERT_MAX_RETRIES**: Number of retries for failed API calls (default: 3)
- **ALERT_RETRY_BACKOFF_SECONDS**: Seconds to wait between retries (default: 30)
- **ALERT_COOLDOWN_MINUTES**: Minimum time before sending another alert for the same monitor (default: 30)

## Development

### Project Structure

```
.
├── cmd/server/           # Application entry point
├── internal/
│   ├── app/             # Application setup and lifecycle
│   ├── config/          # Configuration management
│   ├── domain/          # Domain models (shared)
│   ├── httpclient/      # HTTP client wrapper
│   ├── middleware/      # HTTP middlewares
│   ├── modules/         # Feature modules
│   │   ├── monitor/     # Monitor management
│   │   ├── incident/    # Incident tracking
│   │   └── metrics/     # Statistics
│   ├── scheduler/       # Job scheduling
│   ├── storage/         # Database connection
│   ├── utils/           # Utility functions
│   └── worker/          # Background workers
├── sql/                 # Database schemas
├── Makefile            # Build automation
└── README.md
```

### Running Tests

```bash
make test

# With coverage
make test-coverage
```

### Code Formatting

```bash
make fmt
```

### Available Make Commands

```bash
make help           # Show available commands
make deps           # Install dependencies
make build          # Build application
make run            # Run application
make dev            # Run with hot reload
make test           # Run tests
make clean          # Clean build artifacts
make fmt            # Format code
make docker-build   # Build Docker image
make docker-run     # Run Docker container
make docker-push    # Push Docker image
make docker-release # Release Docker image
make docker-release IMAGE_TAG=v1.0.0 # Release specific tag

```

## How It Works

1. **Monitor Creation**: User creates monitors via API
2. **Scheduler Loads**: Scheduler loads active monitors on startup
3. **Periodic Checks**: Cron jobs trigger HTTP checks at configured intervals
4. **Result Storage**: Each check result is stored in `monitor_checks`
5. **Incident Detection**: 
   - DOWN → Creates incident
   - UP (after DOWN) → Resolves incident
6. **Stats Update**: After each check, statistics are recalculated
7. **API Access**: Users query monitors, checks, incidents, and stats via API

## Performance Considerations

- **Connection Pooling**: PostgreSQL connection pool for efficient DB access
- **Concurrent Checks**: Worker pool for parallel monitoring
- **Indexed Queries**: Database indexes on frequently queried columns
- **Cached Stats**: Pre-calculated statistics for fast API responses
- **Efficient Scheduling**: Cron-based scheduling with minimal overhead

## Error Handling

- Comprehensive error logging
- Graceful degradation
- Timeout protection
- Database transaction safety
- Request validation

## Security Notes

⚠️ **Important**: The `.env` file contains sensitive credentials. In production:

1. Use environment variables instead of `.env` files
2. Enable SSL/TLS for database connections
3. Implement authentication/authorization
4. Use secrets management (Vault, AWS Secrets Manager, etc.)
5. Rotate database credentials regularly

## License

MIT License - Feel free to use this project for personal or commercial purposes.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Support

For issues, questions, or contributions, please open an issue on GitHub.