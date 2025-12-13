-- Required for UUID generation
create extension if not exists "pgcrypto";


-- Table to store monitored endpoints
create table monitors (
    -- Unique ID for each monitor
    id uuid primary key default gen_random_uuid(),

    -- Owner of the monitor (Supabase auth user)
    -- Can be NULL if auth is not implemented yet
    user_id uuid null,

    -- Human-friendly name (e.g. "Homepage", "API Health")
    name text not null,

    -- The actual URL to monitor
    url text not null,

    -- HTTP method used for the check
    -- Mostly GET, but POST/HEAD are possible
    method text not null default 'GET',

    -- Which HTTP status code means "UP"
    -- Example: 200, 204, 301
    expected_status int not null default 200,

    -- How often this endpoint is checked (in seconds)
    -- Example: 30, 60, 300
    interval_seconds int not null default 60,

    -- Max time to wait before declaring failure
    timeout_seconds int not null default 10,

    -- Allows pausing monitoring without deleting
    is_active boolean not null default true,

    -- Creation time
    created_at timestamptz not null default now(),

    -- Updated automatically when monitor changes
    updated_at timestamptz not null default now()
);

-- Defines WHAT you are monitoring
    -- Configuration table
    -- Does NOT store results (important!)




-- Table to store individual monitor checks
create table monitor_checks (
    -- Auto-increment ID (lots of rows)
    id bigserial primary key,

    -- Which monitor this check belongs to
    monitor_id uuid not null references monitors(id) on delete cascade,

    -- HTTP response status (200, 500, etc.)
    status_code int,

    -- How long the request took (milliseconds)
    response_time_ms int,

    -- TRUE = UP, FALSE = DOWN
    is_up boolean not null,

    -- Error message if request failed
    -- Example: timeout, DNS failure
    error_message text,

    -- When this check was performed
    checked_at timestamptz not null default now()
);

-- Why this table exists
  --  Stores every ping
  --  Used to:
  --  Calculate uptime %
  --  Draw graphs
  --  Debug failures




-- Table to store incidents (downtime events)
create table incidents (
    -- Incident ID
    id bigserial primary key,

    -- Which monitor was affected
    monitor_id uuid not null references monitors(id) on delete cascade,

    -- When downtime started
    started_at timestamptz not null,

    -- When service recovered
    -- NULL means still down
    resolved_at timestamptz,

    -- Optional reason (timeout, 500 error, etc.)
    cause text,

    -- TRUE if resolved, FALSE if ongoing
    is_resolved boolean not null default false
);

-- Why this table exists
   -- Avoids scanning millions of monitor_checks
   -- Represents meaningful downtime events
   -- Used for:
   -- Incident history
   -- Alerts
   -- Status pages
   -- Without this table, you’d have to infer incidents repeatedly.

-- Table to store aggregated monitor statistics (cached)
create table monitor_stats (
    -- One-to-one with monitors
    monitor_id uuid primary key references monitors(id) on delete cascade,

    -- Uptime percentage (last 24h / 7d / 30d)
    uptime_percentage numeric(5,2) not null default 100.00,

    -- Average response time
    avg_response_time_ms int,

    -- Last time this monitor was checked
    last_checked_at timestamptz,

    -- Last known state
    -- TRUE = UP, FALSE = DOWN
    last_status boolean
);
-- Why this table exists
   -- Caches frequently needed stats
   -- Improves dashboard performance
   -- Avoids expensive calculations on-the-fly


monitors
  |
  v
monitor_checks (many rows)
  |
  |-- if UP → DOWN → create incident
  |-- if DOWN → UP → resolve incident
  |
  v
monitor_stats (updated each check)

