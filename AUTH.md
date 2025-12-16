# Authentication Guide

This document explains the hybrid authentication system implemented in the Uptime Monitor API.

## Overview

The API supports **two authentication methods**:

1. **Clerk JWT** - For frontend/dashboard users (Next.js app)
2. **API Keys** - For B2B/API-as-a-Service customers

Both methods normalize to a single `user_id` that owns all monitors, incidents, and data.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Client Types                         │
├─────────────────────────────────────────────────────────┤
│  Frontend (Browser/App)    │    API Customers (B2B)     │
│  └─ Clerk JWT Auth         │    └─ API Key Auth         │
└──────────────┬──────────────┴────────────┬──────────────┘
               │                           │
               └────────────┬──────────────┘
                            │
┌───────────────────────────▼───────────────────────────┐
│            Auth Middleware (Hybrid)                   │
│  • Checks Authorization: Bearer <token>               │
│  • Checks X-API-Key: <key>                            │
│  • Normalizes to user_id                              │
└───────────────────────────┬───────────────────────────┘
                            │
┌───────────────────────────▼───────────────────────────┐
│               AuthContext                             │
│  {                                                    │
│    user_id: "clerk_abc123" or "api_xyz789"            │
│    user_type: "clerk" or "api_customer"               │
│    scopes: ["*"] or ["monitors:read", ...]            │
│  }                                                    │
└───────────────────────────┬───────────────────────────┘
                            │
┌───────────────────────────▼───────────────────────────┐
│         Business Logic (Services)                     │
│  • All queries scoped by user_id                      │
│  • WHERE user_id = $1                                 │
└───────────────────────────┬───────────────────────────┘
                            │
┌───────────────────────────▼───────────────────────────┐
│              Database (PostgreSQL)                    │
│  • monitors.user_id (text)                            │
│  • api_keys.user_id (text)                            │
│  • users table (optional metadata)                    │
└───────────────────────────────────────────────────────┘
```

## Method 1: Clerk JWT Authentication

### How It Works

1. User logs in via Clerk on Next.js frontend
2. Frontend receives JWT token from Clerk
3. Frontend sends requests with `Authorization: Bearer <token>` header
4. Backend verifies JWT using Clerk's JWKS public keys
5. Backend extracts Clerk user ID and creates/updates user in database
6. Request proceeds with `user_id` from Clerk

### Configuration

Add to `.env`:

```bash
CLERK_SECRET_KEY=sk_test_...
CLERK_PUBLISHABLE_KEY=pk_test_...
CLERK_JWKS_URL=https://your-clerk-domain.clerk.accounts.dev/.well-known/jwks.json
```

### Frontend Usage (Next.js)

```typescript
// Get token from Clerk
import { useAuth } from '@clerk/nextjs';

const { getToken } = useAuth();

// Make API request
const token = await getToken();
const response = await fetch('http://localhost:5000/api/v1/monitors', {
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  },
});
```

### Token Flow

```
1. User logs in → Clerk generates JWT
2. JWT contains: { sub: "user_123", email: "user@example.com", ... }
3. Backend verifies signature using JWKS
4. Backend extracts user_id from 'sub' claim
5. Backend creates/updates user record
6. Request proceeds with user_id = "user_123"
```

## Method 2: API Key Authentication

### How It Works

1. Authenticated user creates API key via dashboard
2. Backend generates unique key (e.g., `sk_live_abc123...`)
3. Backend hashes key with SHA-256 and stores hash only
4. User receives raw key once (never shown again)
5. API customer sends requests with `X-API-Key: <key>` header
6. Backend hashes provided key and looks up in database
7. Request proceeds with `user_id` from API key record

### Creating an API Key

**Endpoint:** `POST /api/v1/auth/api-keys`

**Headers:**
```
Authorization: Bearer <clerk_jwt>
```

**Request:**
```json
{
  "name": "Production Server",
  "scopes": ["monitors:read", "monitors:write", "incidents:read"],
  "rate_limit_per_hour": 5000,
  "expires_at": "2025-12-31T23:59:59Z"  // optional
}
```

**Response:**
```json
{
  "message": "API key created successfully. Save this key, it won't be shown again.",
  "api_key": {
    "id": "uuid-...",
    "name": "Production Server",
    "raw_key": "sk_live_abc123def456...",  // ⚠️ SAVE THIS - shown only once
    "key_prefix": "sk_live_",
    "scopes": ["monitors:read", "monitors:write", "incidents:read"],
    "rate_limit_per_hour": 5000,
    "is_active": true,
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

### Using an API Key

```bash
curl -X GET http://localhost:5000/api/v1/monitors \
  -H "X-API-Key: sk_live_abc123def456..."
```

### API Key Management

**List Keys:**
```bash
GET /api/v1/auth/api-keys
Authorization: Bearer <clerk_jwt>
```

**Delete Key:**
```bash
DELETE /api/v1/auth/api-keys/:id
Authorization: Bearer <clerk_jwt>
```

**Disable/Enable Key:**
```bash
PUT /api/v1/auth/api-keys/:id/status
Authorization: Bearer <clerk_jwt>
Content-Type: application/json

{
  "is_active": false
}
```

## Scopes and Permissions

### Available Scopes

- `monitors:read` - List and view monitors
- `monitors:write` - Create, update, delete monitors
- `incidents:read` - View incidents
- `stats:read` - View statistics

### Clerk Users

Clerk users automatically get all scopes (`["*"]`)

### API Key Users

API keys can have specific scopes. If no scopes provided, defaults to:
```json
["monitors:read", "monitors:write", "incidents:read", "stats:read"]
```

### Scope Enforcement (Future)

Currently all authenticated users have full access. To implement scope checking:

```go
func RequireScope(scope string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        authCtx := c.Locals("auth").(*domain.AuthContext)
        
        if !hasScope(authCtx.Scopes, scope) {
            return c.Status(403).JSON(fiber.Map{
                "error": "Insufficient permissions",
            })
        }
        
        return c.Next()
    }
}
```

## Database Schema

### users Table

```sql
CREATE TABLE users (
    user_id text PRIMARY KEY,           -- Clerk ID or generated ID
    user_type text NOT NULL,            -- 'clerk' or 'api_customer'
    email text,
    name text,
    clerk_user_id text UNIQUE,          -- Original Clerk ID
    company_name text,                  -- For API customers
    is_active boolean DEFAULT true,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);
```

### api_keys Table

```sql
CREATE TABLE api_keys (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id text NOT NULL,              -- References users.user_id
    key_hash text NOT NULL UNIQUE,      -- SHA-256 hash (never store raw key)
    name text NOT NULL,
    key_prefix text NOT NULL,           -- First 8 chars for identification
    scopes jsonb DEFAULT '[]'::jsonb,
    rate_limit_per_hour int DEFAULT 1000,
    is_active boolean DEFAULT true,
    expires_at timestamptz,
    last_used_at timestamptz,
    total_requests bigint DEFAULT 0,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);
```

### monitors Table (Updated)

```sql
CREATE TABLE monitors (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id text,                       -- Changed from uuid to text
    name text NOT NULL,
    url text NOT NULL,
    -- ... other fields
);
```

## Security Features

### 1. API Key Hashing

```go
// Never store raw keys
rawKey := "sk_live_abc123..."
keyHash := sha256.Sum256([]byte(rawKey))
storedHash := hex.EncodeToString(keyHash[:])

// To verify
providedHash := sha256.Sum256([]byte(providedKey))
isValid := storedHash == hex.EncodeToString(providedHash[:])
```

### 2. JWT Verification

- Fetches JWKS from Clerk
- Verifies RSA signature
- Checks expiration
- Validates issuer

### 3. Rate Limiting (Planned)

Each API key has `rate_limit_per_hour` field for future rate limiting implementation.

### 4. Key Expiration

Optional `expires_at` field allows time-limited keys.

### 5. Audit Trail

- `last_used_at` - Last time key was used
- `total_requests` - Total number of requests made with key

## API Examples

### 1. Create Monitor (Clerk User)

```bash
curl -X POST http://localhost:5000/api/v1/monitors \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Website",
    "url": "https://example.com",
    "method": "GET",
    "expected_status": 200,
    "interval_seconds": 60,
    "timeout_seconds": 10
  }'
```

### 2. Create Monitor (API Key)

```bash
curl -X POST http://localhost:5000/api/v1/monitors \
  -H "X-API-Key: sk_live_abc123..." \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Website",
    "url": "https://example.com",
    "method": "GET",
    "expected_status": 200,
    "interval_seconds": 60,
    "timeout_seconds": 10
  }'
```

### 3. List Monitors

Both methods return only monitors owned by the authenticated user:

```bash
# Clerk
curl http://localhost:5000/api/v1/monitors \
  -H "Authorization: Bearer eyJhbGc..."

# API Key
curl http://localhost:5000/api/v1/monitors \
  -H "X-API-Key: sk_live_abc123..."
```

### 4. Get Current User

```bash
curl http://localhost:5000/api/v1/auth/me \
  -H "Authorization: Bearer eyJhbGc..."
```

Response:
```json
{
  "user_id": "user_123",
  "user_type": "clerk",
  "email": "user@example.com",
  "name": "John Doe",
  "scopes": ["*"]
}
```

## Error Responses

### No Authentication

```json
{
  "error": "Authentication required. Provide either 'Authorization: Bearer <token>' or 'X-API-Key: <key>' header"
}
```
Status: 401

### Invalid JWT

```json
{
  "error": "Invalid or expired token"
}
```
Status: 401

### Invalid API Key

```json
{
  "error": "Invalid API key"
}
```
Status: 401

### Not Found / Unauthorized

When accessing resources belonging to another user:
```json
{
  "error": "Monitor not found"
}
```
Status: 404

## Testing Authentication

### Test with Clerk JWT (cURL)

```bash
# Get token from your frontend or Clerk dashboard
TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."

curl http://localhost:5000/api/v1/auth/me \
  -H "Authorization: Bearer $TOKEN"
```

### Test with API Key

```bash
# Create API key first (requires Clerk auth)
curl -X POST http://localhost:5000/api/v1/auth/api-keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Key"}'

# Use the returned key
API_KEY="sk_live_abc123..."
curl http://localhost:5000/api/v1/monitors \
  -H "X-API-Key: $API_KEY"
```

## Migration Guide

### Updating Existing Data

If you have existing monitors without user_id:

```sql
-- Option 1: Assign to a default user
UPDATE monitors SET user_id = 'default_user' WHERE user_id IS NULL;

-- Option 2: Delete orphaned monitors
DELETE FROM monitors WHERE user_id IS NULL;
```

### Altering user_id Column

Already done in schema:
```sql
ALTER TABLE monitors ALTER COLUMN user_id TYPE text;
```

## Best Practices

### Frontend (Clerk)

1. **Never store JWT permanently** - Always get fresh tokens
2. **Include token in every request**
3. **Handle 401 errors** - Redirect to login
4. **Use HTTPS only**

### API Keys

1. **Store securely** - Use environment variables or secret managers
2. **Rotate regularly** - Create new keys, delete old ones
3. **Use minimal scopes** - Only grant necessary permissions
4. **Set expiration** - Time-limit sensitive keys
5. **Monitor usage** - Check `total_requests` and `last_used_at`

### Backend

1. **Never log raw keys** - Only log key prefixes
2. **Use HTTPS in production**
3. **Implement rate limiting** - Based on `rate_limit_per_hour`
4. **Audit trail** - Log authentication events
5. **Rotate secrets** - Change CLERK_SECRET_KEY if compromised

## Troubleshooting

### "Invalid or expired token"

- Token might be expired (check `exp` claim)
- JWKS URL might be wrong
- Clock skew between servers
- Token from wrong Clerk application

### "Invalid API key"

- Key not active (`is_active = false`)
- Key expired (`expires_at` passed)
- Key deleted
- Wrong key format
- Key hash mismatch

### "Monitor not found" (when monitor exists)

- Monitor belongs to different user
- Check user_id in database vs authenticated user_id

### Clerk JWKS fetch fails

- Check `CLERK_JWKS_URL` is correct
- Ensure server can reach Clerk's servers
- Check firewall/proxy settings

## Future Enhancements

1. **Rate Limiting** - Enforce `rate_limit_per_hour`
2. **Scope Enforcement** - Restrict actions based on scopes
3. **Webhook Signatures** - Sign webhook payloads with API keys
4. **Team Management** - Multiple users per account
5. **OAuth 2.0** - Support OAuth for third-party integrations
6. **API Key Rotation** - Automatic key rotation policies

## Summary

✅ **Hybrid Auth** - Clerk JWT + API Keys
✅ **Single user_id** - All data owned by user_id (text)
✅ **Secure Storage** - Keys hashed with SHA-256
✅ **Scoped Access** - All queries filtered by user_id
✅ **Flexible** - Easy to add more auth methods
✅ **Production-Ready** - Proper error handling and validation

The system allows both dashboard users (Clerk) and API customers (API Keys) to access the same API with their own isolated data, all managed through a single `user_id` column.