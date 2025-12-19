# Security Fixes & Improvements

## Overview

This document details all security fixes and improvements made to the authentication and user lifecycle management system.

## Fixed Security Issues

### 🔒 Issue 1: Orphaned Data on User Deletion

**Problem:**
- Deleting a user only deleted the `users` row
- Monitors, API keys, checks, and incidents remained orphaned
- API keys could still be used after user deletion
- Database integrity compromised

**Solution:**
```sql
-- Added CASCADE DELETE foreign keys
ALTER TABLE monitors 
    ADD CONSTRAINT fk_monitors_user_id 
    FOREIGN KEY (user_id) 
    REFERENCES users(user_id) 
    ON DELETE CASCADE;

ALTER TABLE api_keys 
    ADD CONSTRAINT fk_api_keys_user_id 
    FOREIGN KEY (user_id) 
    REFERENCES users(user_id) 
    ON DELETE CASCADE;
```

**Result:**
- ✅ Deleting user → Automatically deletes monitors
- ✅ Deleting monitors → Automatically deletes checks (existing CASCADE)
- ✅ Deleting monitors → Automatically deletes incidents (existing CASCADE)
- ✅ Deleting user → Automatically deletes API keys
- ✅ Complete cleanup with single DELETE operation

**Cascade Chain:**
```
DELETE user
    ↓
CASCADE DELETE monitors
    ↓
CASCADE DELETE monitor_checks (via monitors)
    ↓
CASCADE DELETE incidents (via monitors)
    ↓
CASCADE DELETE monitor_stats (via monitors)
    
DELETE user
    ↓
CASCADE DELETE api_keys
```

### 🔒 Issue 2: Inactive Users Can Still Access API

**Problem:**
- Disabled/deactivated users could still authenticate
- API keys from inactive users still worked
- No lifecycle enforcement

**Solution:**

**1. Repository-Level Checks:**
```go
// GetAPIKeyByHash now joins with users table
SELECT ak.*, u.is_active as user_is_active
FROM api_keys ak
INNER JOIN users u ON ak.user_id = u.user_id
WHERE ak.key_hash = $1 
  AND ak.is_active = true
  AND u.is_active = true  -- NEW: Check user status
```

**2. Service-Level Checks:**
```go
// VerifyClerkToken
if !user.IsActive {
    return nil, fmt.Errorf("user account is deactivated")
}

// VerifyAPIKey - double check
isActive, err := s.repo.IsUserActive(ctx, apiKey.UserID)
if !isActive {
    return nil, fmt.Errorf("user account is deactivated")
}
```

**3. Database Triggers:**
```sql
-- Automatically disable monitors when user is deactivated
CREATE TRIGGER trigger_disable_user_monitors
    AFTER UPDATE ON users
    FOR EACH ROW
    WHEN (NEW.is_active = FALSE AND OLD.is_active = TRUE)
    EXECUTE FUNCTION disable_user_monitors();

-- Automatically disable API keys when user is deactivated
CREATE TRIGGER trigger_disable_user_api_keys
    AFTER UPDATE ON users
    FOR EACH ROW
    WHEN (NEW.is_active = FALSE AND OLD.is_active = TRUE)
    EXECUTE FUNCTION disable_user_api_keys();
```

**Result:**
- ✅ Deactivated users cannot authenticate
- ✅ API keys from inactive users are rejected
- ✅ Monitors automatically disabled on user deactivation
- ✅ API keys automatically disabled on user deactivation

### 🔒 Issue 3: Missing User Data from Clerk JWT

**Problem:**
- JWT only contains `sub` (user ID)
- Email and name fields remained empty in database
- Poor user experience and missing audit data

**Solution:**

**1. Clerk Backend API Client:**
```go
// New ClerkClient to fetch user details
type ClerkClient struct {
    secretKey  string
    baseURL    string
    httpClient *http.Client
}

func (c *ClerkClient) GetUser(ctx context.Context, userID string) (*ClerkUser, error) {
    // Fetches email, first_name, last_name from Clerk
}
```

**2. Automatic User Enrichment:**
```go
// On first authentication
func (s *Service) VerifyClerkToken(ctx context.Context, token string) (*domain.AuthContext, error) {
    // 1. Verify JWT
    claims, err := s.clerkVerifier.VerifyToken(token)
    
    // 2. Check if user exists
    user, err := s.repo.GetUserByClerkID(ctx, clerkUserID)
    if err == pgx.ErrNoRows {
        // 3. User doesn't exist - fetch from Clerk API
        user, err = s.createUserFromClerk(ctx, clerkUserID)
    }
    
    // 4. If email/name empty, enrich from Clerk
    if user.Email == "" || user.Name == "" {
        s.enrichUserDataFromClerk(ctx, user)
    }
}
```

**3. Name Handling:**
```go
// GetFullName combines first and last name
func (u *ClerkUser) GetFullName() string {
    firstName := strings.TrimSpace(u.FirstName)
    lastName := strings.TrimSpace(u.LastName)
    
    if firstName != "" && lastName != "" {
        return firstName + " " + lastName
    }
    if firstName != "" {
        return firstName
    }
    if lastName != "" {
        return lastName
    }
    return ""
}

// Fallback to email if name is empty
if user.Name == "" && user.Email != "" {
    user.Name = user.Email
}
```

**Result:**
- ✅ User data automatically fetched from Clerk on first login
- ✅ Email and name populated in database
- ✅ Graceful handling of missing last names
- ✅ Fallback to email if name unavailable
- ✅ Automatic enrichment if data missing

### 🔒 Issue 4: API Key Performance Issues

**Problem:**
- `total_requests` updated on every API call
- Row locking on high-traffic API keys
- Potential performance bottleneck

**Solution:**

**1. Asynchronous Best-Effort Updates:**
```go
// Update usage tracking (async, don't block on errors)
go func() {
    updateCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    _ = s.repo.UpdateAPIKeyUsage(updateCtx, apiKey.ID)
}()
```

**2. Removed Transaction Blocks:**
```go
// Best-effort update, don't block on errors
func (r *Repository) UpdateAPIKeyUsage(ctx context.Context, keyID string) error {
    query := `
        UPDATE api_keys
        SET last_used_at = NOW(),
            total_requests = total_requests + 1
        WHERE id = $1
    `
    _, err := r.db.Exec(ctx, query, keyID)
    return err // Don't fail request if this fails
}
```

**3. Partial Index for Active Keys:**
```sql
CREATE INDEX idx_api_keys_key_hash 
ON api_keys(key_hash) 
WHERE is_active = true;
```

**Result:**
- ✅ Non-blocking usage tracking
- ✅ No request failures if tracking fails
- ✅ 2-second timeout for updates
- ✅ Minimal performance impact
- ✅ Acceptable data loss for non-critical metrics

### 🔒 Issue 5: Missing Uniqueness Constraints

**Problem:**
- Duplicate API keys possible (security risk)
- Duplicate monitor names per user (confusing UX)

**Solution:**

**1. API Key Hash Uniqueness:**
```sql
ALTER TABLE api_keys 
    ADD CONSTRAINT uq_api_keys_key_hash 
    UNIQUE (key_hash);
```

**2. Monitor Name Uniqueness Per User:**
```sql
ALTER TABLE monitors 
    ADD CONSTRAINT uq_monitors_user_name 
    UNIQUE (user_id, name);
```

**3. API Key Name Uniqueness Per User:**
```sql
ALTER TABLE api_keys 
    ADD CONSTRAINT uq_api_keys_user_name 
    UNIQUE (user_id, name);
```

**Result:**
- ✅ Impossible to create duplicate API keys
- ✅ Better UX with unique monitor names per user
- ✅ Better UX with unique API key names per user
- ✅ Database-level constraint enforcement

### 🔒 Issue 6: Missing Performance Indexes

**Problem:**
- Large tables without indexes slow down queries
- User filtering on monitors inefficient
- API key lookups slow at scale

**Solution:**

**1. User-Scoped Queries:**
```sql
-- Compound index for most common query pattern
CREATE INDEX idx_monitors_user_id_is_active 
    ON monitors(user_id, is_active);

-- Individual indexes
CREATE INDEX idx_monitors_user_id ON monitors(user_id);
CREATE INDEX idx_monitors_is_active ON monitors(is_active);
```

**2. API Key Lookups:**
```sql
-- Partial index for active keys only
CREATE INDEX idx_api_keys_key_hash 
    ON api_keys(key_hash) 
    WHERE is_active = true;

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
```

**3. Time-Series Data:**
```sql
-- For recent checks queries
CREATE INDEX idx_monitor_checks_monitor_id_checked_at 
    ON monitor_checks(monitor_id, checked_at DESC);

-- For incident queries
CREATE INDEX idx_incidents_monitor_id_is_resolved 
    ON incidents(monitor_id, is_resolved);
```

**4. User Lookups:**
```sql
CREATE INDEX idx_users_clerk_user_id ON users(clerk_user_id);
CREATE INDEX idx_users_is_active ON users(is_active);
CREATE INDEX idx_users_email ON users(email);
```

**Result:**
- ✅ Fast user-scoped monitor queries
- ✅ Efficient API key verification
- ✅ Quick historical data retrieval
- ✅ Optimized for common access patterns

## New API Endpoints

### User Account Management

**Delete Account (Hard Delete):**
```bash
DELETE /api/v1/auth/account
Authorization: Bearer <token>
Content-Type: application/json

{
  "confirm": "DELETE"
}
```

Response:
```json
{
  "message": "Account and all associated data deleted successfully"
}
```

**Deactivate Account (Soft Delete):**
```bash
POST /api/v1/auth/account/deactivate
Authorization: Bearer <token>
```

Response:
```json
{
  "message": "Account deactivated successfully. All monitors and API keys have been disabled."
}
```

## Database Changes Summary

### New Constraints

| Table | Constraint | Type | Purpose |
|-------|-----------|------|---------|
| monitors | fk_monitors_user_id | FK | CASCADE DELETE |
| api_keys | fk_api_keys_user_id | FK | CASCADE DELETE |
| api_keys | uq_api_keys_key_hash | UNIQUE | Prevent duplicate keys |
| api_keys | uq_api_keys_user_name | UNIQUE | Prevent duplicate names |
| monitors | uq_monitors_user_name | UNIQUE | Prevent duplicate names |

### New Indexes

| Table | Index | Purpose |
|-------|-------|---------|
| monitors | idx_monitors_user_id_is_active | User queries |
| api_keys | idx_api_keys_key_hash | Key verification |
| monitor_checks | idx_monitor_checks_monitor_id_checked_at | Recent checks |
| incidents | idx_incidents_monitor_id_is_resolved | Incident queries |
| users | idx_users_clerk_user_id | Clerk lookup |
| users | idx_users_is_active | Active user filter |

### New Triggers

| Trigger | Purpose |
|---------|---------|
| trigger_disable_user_monitors | Auto-disable monitors on user deactivation |
| trigger_disable_user_api_keys | Auto-disable API keys on user deactivation |

### New Views

| View | Purpose |
|------|---------|
| user_stats | Aggregate user statistics (monitors, API keys) |

## Migration Steps

### 1. Backup Data (Recommended)

```sql
CREATE TABLE monitors_backup AS SELECT * FROM monitors;
CREATE TABLE api_keys_backup AS SELECT * FROM api_keys;
CREATE TABLE users_backup AS SELECT * FROM users;
```

### 2. Run Migration Script

```bash
psql $DATABASE_URL -f sql/migration_add_constraints.sql
```

### 3. Verify Migration

```sql
-- Check foreign keys
SELECT * FROM information_schema.table_constraints
WHERE table_name IN ('monitors', 'api_keys')
AND constraint_type = 'FOREIGN KEY';

-- Check indexes
SELECT tablename, indexname FROM pg_indexes
WHERE tablename IN ('monitors', 'api_keys', 'users');

-- Check user stats view
SELECT * FROM user_stats;
```

### 4. Test Cascade Delete

```sql
-- Test in dev/staging first!
BEGIN;

-- Create test user
INSERT INTO users (user_id, user_type, email, name, is_active)
VALUES ('test_delete', 'clerk', 'test@test.com', 'Test User', true);

-- Create test monitor
INSERT INTO monitors (user_id, name, url, method, expected_status, interval_seconds, timeout_seconds)
VALUES ('test_delete', 'Test Monitor', 'https://example.com', 'GET', 200, 60, 10);

-- Create test API key
INSERT INTO api_keys (user_id, key_hash, name, key_prefix, is_active)
VALUES ('test_delete', 'test_hash', 'Test Key', 'sk_test_', true);

-- Verify created
SELECT 
    (SELECT COUNT(*) FROM monitors WHERE user_id = 'test_delete') as monitors,
    (SELECT COUNT(*) FROM api_keys WHERE user_id = 'test_delete') as api_keys;

-- Delete user
DELETE FROM users WHERE user_id = 'test_delete';

-- Verify cascade
SELECT 
    (SELECT COUNT(*) FROM monitors WHERE user_id = 'test_delete') as monitors,
    (SELECT COUNT(*) FROM api_keys WHERE user_id = 'test_delete') as api_keys;
-- Should both be 0

ROLLBACK; -- Or COMMIT if satisfied
```

## Testing Checklist

### Authentication Tests

- [x] Clerk JWT verification works
- [x] User data fetched from Clerk API
- [x] Email and name populated
- [x] Inactive users cannot authenticate
- [x] API keys from inactive users rejected

### Cascade Delete Tests

- [x] Delete user → monitors deleted
- [x] Delete user → API keys deleted
- [x] Delete user → checks deleted (via monitors)
- [x] Delete user → incidents deleted (via monitors)

### Deactivation Tests

- [x] Deactivate user → monitors disabled
- [x] Deactivate user → API keys disabled
- [x] Deactivated user cannot login
- [x] API keys from deactivated user rejected

### Uniqueness Tests

- [x] Cannot create duplicate API keys
- [x] Cannot create duplicate monitor names per user
- [x] Cannot create duplicate API key names per user

### Performance Tests

- [x] User-scoped queries use indexes
- [x] API key lookup uses partial index
- [x] Monitor checks query uses index
- [x] Usage tracking doesn't block requests

## Security Best Practices

### 1. User Lifecycle

✅ Always check `is_active` status
✅ Use CASCADE DELETE for data cleanup
✅ Implement soft delete for audit requirements
✅ Auto-disable resources on user deactivation

### 2. API Key Management

✅ Hash keys with SHA-256
✅ Never log raw keys
✅ Implement key rotation policies
✅ Set expiration dates
✅ Monitor usage with `total_requests`

### 3. Data Access

✅ Always scope queries by `user_id`
✅ Use foreign keys for referential integrity
✅ Implement uniqueness constraints
✅ Add indexes for performance

### 4. Authentication

✅ Verify JWT signatures
✅ Check token expiration
✅ Fetch user data from Clerk API
✅ Implement rate limiting (future)

## Rollback Plan

If issues occur:

```sql
-- 1. Drop new constraints
ALTER TABLE monitors DROP CONSTRAINT IF EXISTS fk_monitors_user_id;
ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS fk_api_keys_user_id;

-- 2. Restore from backup
TRUNCATE monitors;
INSERT INTO monitors SELECT * FROM monitors_backup;

TRUNCATE api_keys;
INSERT INTO api_keys SELECT * FROM api_keys_backup;

-- 3. Drop indexes if causing issues
DROP INDEX IF EXISTS idx_monitors_user_id_is_active;
-- ... other indexes
```

## Monitoring

### Key Metrics to Track

1. **Authentication Failures**
   - Track 401 errors
   - Monitor inactive user attempts

2. **Cascade Deletes**
   - Log user deletions
   - Track cascade counts

3. **API Key Usage**
   - Monitor `total_requests`
   - Track key expiration

4. **Database Performance**
   - Query execution times
   - Index usage statistics

### Useful Queries

```sql
-- Active users with resources
SELECT * FROM user_stats WHERE is_active = true;

-- Inactive users still with data (should be 0)
SELECT u.user_id, COUNT(m.id) as monitors, COUNT(ak.id) as api_keys
FROM users u
LEFT JOIN monitors m ON u.user_id = m.user_id
LEFT JOIN api_keys ak ON u.user_id = ak.user_id
WHERE u.is_active = false
GROUP BY u.user_id
HAVING COUNT(m.id) > 0 OR COUNT(ak.id) > 0;

-- Expired API keys still active
SELECT * FROM api_keys 
WHERE is_active = true 
AND expires_at < NOW();
```

## Summary

All security issues have been fixed:

✅ **Orphaned Data** - Fixed with CASCADE DELETE
✅ **Inactive User Access** - Fixed with active status checks
✅ **Missing User Data** - Fixed with Clerk API integration
✅ **Performance Issues** - Fixed with async updates
✅ **Duplicate Keys** - Fixed with uniqueness constraints
✅ **Missing Indexes** - Fixed with comprehensive indexing

The system now provides:
- Complete data cleanup on user deletion
- Proper lifecycle management
- Rich user data from Clerk
- High performance at scale
- Data integrity guarantees