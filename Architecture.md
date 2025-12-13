# Architecture Documentation

## System Overview

The Uptime Monitor is a distributed monitoring system designed to track website and API endpoint availability. The architecture follows Clean Architecture principles with clear separation of concerns and dependency inversion.

## High-Level Architecture

```
┌──────────────────────────────────────────────────────────┐
│                      Client Layer                        │
│              (HTTP Clients, Web Apps, CLI)               │
└─────────────────────┬────────────────────────────────────┘
                      │ HTTP/REST
┌─────────────────────▼────────────────────────────────────┐
│                    API Gateway (Fiber)                   │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐   │
│  │ Middleware  │  │   Routing    │  │ Error Handler  │   │
│  └─────────────┘  └──────────────┘  └────────────────┘   │
└─────────────────────┬────────────────────────────────────┘
                      │
┌─────────────────────▼────────────────────────────────────┐
│                   Application Layer                      │
│  ┌──────────────────────────────────────────────────┐    │
│  │              Handler Layer                       │    │
│  │  ┌────────────┐  ┌────────────┐  ┌───────────┐ │      │
│  │  │  Monitor   │  │  Incident  │  │  Metrics  │ │      │
│  │  │  Handler   │  │  Handler   │  │  Handler  │ │      │
│  │  └──────┬─────┘  └─────┬──────┘  └─────┬─────┘ │      │
│  └─────────┼──────────────┼───────────────┼────────┘     │
│            │              │               │              │
│  ┌─────────▼──────────────▼───────────────▼────────┐     │
│  │              Service Layer                      │     │
│  │  ┌────────────┐  ┌────────────┐  ┌───────────┐ │      │
│  │  │  Monitor   │  │  Incident  │  │  Metrics  │ │      │
│  │  │  Service   │  │  Service   │  │  Service  │ │      │
│  │  └──────┬─────┘  └─────┬──────┘  └─────┬─────┘ │      │
│  └─────────┼──────────────┼───────────────┼────────┘     │
│            │              │               │              │
│  ┌─────────▼──────────────▼───────────────▼────────┐     │
│  │            Repository Layer                     │     │
│  │  ┌────────────┐  ┌────────────┐  ┌───────────┐  │     │
│  │  │  Monitor   │  │  Incident  │  │  Metrics  │  │     │
│  │  │    Repo    │  │    Repo    │  │    Repo   │  │     │
│  │  └──────┬─────┘  └─────┬──────┘  └─────┬─────┘  │     │
│  └─────────┼──────────────┼───────────────┼────────┘     │
└────────────┼──────────────┼───────────────┼─────────────--
             │              │               │
┌────────────▼──────────────▼───────────────▼─────────────┐
│                   Data Layer                             │
│              PostgreSQL (Supabase)                       │
│  ┌──────────┐  ┌────────────────┐  ┌─────────────────┐ │
│  │ monitors │  │ monitor_checks │  │    incidents    │ │
│  └──────────┘  └────────────────┘  └─────────────────┘ │
│  ┌──────────────────┐                                   │
│  │  monitor_stats   │                                   │
│  └──────────────────┘                                   │
└──────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────┐
│               Background Processing Layer                 │
│  ┌────────────────────────────────────────────────────┐  │
│  │              Scheduler (Cron)                      │  │
│  │  - Loads active monitors                          │  │
│  │  - Schedules periodic checks                      │  │
│  │  - Manages job lifecycle                          │  │
│  └────────────────┬───────────────────────────────────┘  │
│                   │                                       │
│  ┌────────────────▼───────────────────────────────────┐  │
│  │              Monitor Check Jobs                    │  │
│  │  - HTTP health checks                             │  │
│  │  - Result storage                                 │  │
│  │  - Incident detection                             │  │
│  │  - Stats update                                   │  │
│  └────────────────┬───────────────────────────────────┘  │
│                   │                                       │
│  ┌────────────────▼───────────────────────────────────┐  │
│  │              HTTP Checker Worker                   │  │
│  │  - HTTP client with timeouts                      │  │
│  │  - Response validation                            │  │
│  │  - Error handling                                 │  │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

## Core Components

### 1. API Layer

**Purpose**: Handle HTTP requests and responses

**Components**:
- **Fiber Framework**: High-performance HTTP server
- **Middleware Stack**:
  - Request Logger: Logs all incoming requests
  - Error Handler: Centralized error handling
  - CORS: Cross-origin resource sharing
  - Recovery: Panic recovery

**Flow**:
```
Request → Middleware → Route Handler → Response
```

### 2. Handler Layer

**Location**: `internal/modules/*/handler.go`

**Responsibilities**:
- Parse and validate HTTP requests
- Call appropriate service methods
- Format responses
- Handle HTTP-specific errors

**Example**:
```go
func (h *Handler) Create(c *fiber.Ctx) error {
    var req CreateRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(...)
    }
    
    result, err := h.service.Create(c.Context(), &req)
    if err != nil {
        return c.Status(400).JSON(...)
    }
    
    return c.Status(201).JSON(result)
}
```

### 3. Service Layer

**Location**: `internal/modules/*/service.go`

**Responsibilities**:
- Business logic implementation
- Data validation
- Orchestrate multiple repositories
- Transaction coordination

**Key Services**:

- **MonitorService**: CRUD operations for monitors
- **IncidentService**: Incident lifecycle management
- **MetricsService**: Statistics calculation and aggregation

### 4. Repository Layer

**Location**: `internal/modules/*/repository.go`

**Responsibilities**:
- Database queries (CRUD)
- Data persistence
- Query optimization
- Transaction management

**Pattern**:
```go
type Repository struct {
    db *pgxpool.Pool
}

func (r *Repository) Create(ctx context.Context, data *Data) error {
    query := `INSERT INTO table ...`
    _, err := r.db.Exec(ctx, query, data)
    return err
}
```

### 5. Domain Layer

**Location**: `internal/domain/`

**Purpose**: Prevent circular dependencies

**Contains**:
- Data structures
- Domain models
- DTOs (Data Transfer Objects)
- Request/Response types

**Why Separate**:
- Shared across all layers
- No dependencies on other internal packages
- Clean dependency graph

### 6. Scheduler

**Location**: `internal/scheduler/`

**Components**:

**Scheduler** (`scheduler.go`):
- Manages cron jobs
- Loads monitors on startup
- Add/Remove/Update jobs dynamically

**MonitorCheckJob** (`monitor_job.go`):
- Executes single monitor check
- Stores results
- Detects and manages incidents
- Updates statistics

**Flow**:
```
Scheduler.Start()
   ↓
Load Active Monitors
   ↓
For Each Monitor → Create Cron Job
   ↓
Every N seconds → Execute Check Job
   ↓
├─→ HTTP Check
├─→ Store Result
├─→ Detect/Resolve Incidents
└─→ Update Stats
```

### 7. Worker Layer

**Location**: `internal/worker/`

**HTTPChecker**:
- Performs HTTP requests
- Measures response time
- Validates status codes
- Handles timeouts and errors

**Features**:
- Configurable timeouts
- Connection pooling
- Retry logic (future)
- User-Agent headers

## Data Flow

### Monitor Check Flow

```
┌─────────────────────────────────────────────────────────┐
│ 1. Cron Trigger (Every N seconds)                       │
└────────────────┬────────────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────────────┐
│ 2. MonitorCheckJob.Run()                                │
│    - Get monitor configuration                          │
│    - Prepare check context                              │
└────────────────┬────────────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────────────┐
│ 3. HTTPChecker.Check(monitor)                           │
│    - Create HTTP request                                │
│    - Execute with timeout                               │
│    - Measure response time                              │
│    - Validate status code                               │
└────────────────┬────────────────────────────────────────┘
                 │
                 ├─→ Success: status_code, response_time
                 └─→ Failure: error_message
                 │
┌────────────────▼────────────────────────────────────────┐
│ 4. Save Check Result                                    │
│    INSERT INTO monitor_checks (...)                     │
└────────────────┬────────────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────────────┐
│ 5. Incident Detection                                   │
│    ┌──────────────────────────────────────────────┐    │
│    │ Was DOWN, Now UP → Resolve Incident          │    │
│    │ Was UP, Now DOWN → Create Incident           │    │
│    └──────────────────────────────────────────────┘    │
└────────────────┬────────────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────────────┐
│ 6. Update Statistics                                    │
│    - Calculate uptime percentage                        │
│    - Average response time                              │
│    - Update monitor_stats table                         │
└─────────────────────────────────────────────────────────┘
```

### API Request Flow

```
Client Request
    ↓
Middleware (Logger, CORS, etc.)
    ↓
Route Handler
    ↓
Validation
    ↓
Service Layer (Business Logic)
    ↓
Repository Layer (Database)
    ↓
PostgreSQL
    ↓
Response back up the chain
    ↓
JSON Response to Client
```

## Database Design

### Tables and Relationships

```
monitors (1) ────────→ (N) monitor_checks
    │
    ├──────────────────→ (N) incidents
    │
    └──────────────────→ (1) monitor_stats
```

### Table Purposes

**monitors**:
- Configuration storage
- What to monitor and how

**monitor_checks**:
- Historical check results
- Time-series data
- Used for uptime calculations

**incidents**:
- Downtime tracking
- Aggregated failure events
- Reduces need to scan all checks

**monitor_stats**:
- Pre-calculated metrics
- Fast API responses
- Updated after each check

### Query Patterns

**High-frequency**:
```sql
-- Check storage (every monitor interval)
INSERT INTO monitor_checks (...)

-- Stats update (every monitor interval)
UPDATE monitor_stats SET ...

-- Get recent checks (API)
SELECT * FROM monitor_checks 
WHERE monitor_id = ? 
ORDER BY checked_at DESC 
LIMIT 50
```

**Medium-frequency**:
```sql
-- List monitors (API)
SELECT * FROM monitors

-- Get monitor stats (API)
SELECT * FROM monitor_stats WHERE monitor_id = ?

-- List incidents (API)
SELECT * FROM incidents WHERE monitor_id = ?
```

## Configuration Management

### Environment Variables

**Location**: `.env` file (development) or system environment (production)

**Categories**:

1. **Server**: PORT, timeouts, environment
2. **Database**: Connection string, pool size, SSL
3. **Monitor**: Worker pool, retries, timeouts

**Loading**:
```
.env file → godotenv → os.Getenv → Config struct
```

### Validation

All configuration values are validated on startup:
- Type checking (int, duration, etc.)
- Range validation (min/max values)
- Required vs optional

## Concurrency Model

### Connection Pooling

```
pgxpool.Pool
├─ MaxConns: 25
├─ MinConns: 5
├─ MaxConnLifetime: 1 hour
└─ MaxConnIdleTime: 30 minutes
```

### HTTP Client

```
http.Client
├─ Timeout: Configurable per monitor
├─ MaxIdleConns: 100
├─ MaxIdleConnsPerHost: 10
└─ IdleConnTimeout: 90 seconds
```

### Scheduler Concurrency

- Each monitor check runs in its own goroutine
- Cron scheduler manages timing
- No explicit worker pool (cron handles this)
- Context-based cancellation

## Error Handling Strategy

### Layers

1. **Worker Layer**: Catch HTTP errors, timeouts
2. **Repository Layer**: Handle database errors
3. **Service Layer**: Validate business logic
4. **Handler Layer**: Return appropriate HTTP status

### Error Types

- **Validation Errors**: 400 Bad Request
- **Not Found**: 404 Not Found
- **Database Errors**: 500 Internal Server Error
- **Timeout Errors**: Stored in check result

### Logging

All errors are logged with context:
```
log.Printf("Error: %v | Path: %s | Method: %s", err, path, method)
```

## Security Considerations

### Current State

- SSL/TLS for database (Supabase)
- Environment variable for secrets
- No authentication (must be added)

### Recommendations

1. **Authentication**:
   - JWT tokens
   - API keys
   - OAuth 2.0

2. **Authorization**:
   - User-based monitor ownership
   - Role-based access control

3. **Rate Limiting**:
   - Per-user API limits
   - Prevent abuse

4. **Input Validation**:
   - Already implemented
   - Continue to validate all inputs

5. **Database**:
   - SSL/TLS enforced
   - Parameterized queries (pgx handles this)
   - Connection pooling

## Scalability

### Current Limitations

- Single instance (no horizontal scaling yet)
- In-memory scheduler state
- Synchronous stats updates

### Future Improvements

1. **Horizontal Scaling**:
   - Distributed cron (e.g., with Redis)
   - Stateless API servers
   - Load balancer

2. **Database Optimization**:
   - Read replicas
   - Partitioning monitor_checks by date
   - Archival strategy

3. **Caching**:
   - Redis for stats
   - Reduce database load

4. **Message Queue**:
   - Async check processing
   - Better failure handling

## Monitoring and Observability

### Current Logging

- Request/response logging
- Error logging
- Job execution logging

### Future Additions

1. **Metrics**:
   - Prometheus metrics
   - Check duration
   - Error rates

2. **Tracing**:
   - OpenTelemetry
   - Distributed tracing

3. **Alerting**:
   - Email notifications
   - Webhook callbacks
   - Slack integration

## Testing Strategy

### Unit Tests

- Service layer logic
- Validation functions
- Utility functions

### Integration Tests

- Database operations
- API endpoints
- Scheduler jobs

### Load Tests

- Concurrent monitor checks
- API throughput
- Database performance

## Deployment

### Requirements

- Go 1.21+
- PostgreSQL (Supabase)
- Linux/Unix environment

### Process

1. Build binary: `make build`
2. Set environment variables
3. Run migrations
4. Start server: `./bin/server`

### Docker (Future)

```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o server cmd/server/main.go
CMD ["./server"]
```

## Maintenance

### Database Cleanup

Regularly archive old `monitor_checks`:
```sql
DELETE FROM monitor_checks 
WHERE checked_at < NOW() - INTERVAL '90 days';
```

### Monitoring

- Check database connection health
- Monitor scheduler job execution
- Track API error rates

## Future Enhancements

1. **Multi-region Checks**: Check from multiple locations
2. **SSL Certificate Monitoring**: Alert on expiration
3. **Custom Assertions**: Content validation, headers
4. **Status Page**: Public uptime page
5. **Notifications**: Email, SMS, Webhook
6. **Advanced Metrics**: P50, P95, P99 latency

## References

- [Fiber Documentation](https://docs.gofiber.io/)
- [pgx Documentation](https://pkg.go.dev/github.com/jackc/pgx/v5)
- [Cron Documentation](https://pkg.go.dev/github.com/robfig/cron/v3)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)