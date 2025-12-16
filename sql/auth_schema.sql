-- API Keys table for API-as-a-Service customers
create table if not exists api_keys (
    id uuid primary key default gen_random_uuid(),
    
    -- Owner of this API key (maps to user_id)
    user_id text not null,
    
    -- Hashed API key (never store raw keys)
    key_hash text not null unique,
    
    -- Human-readable name for the key
    name text not null,
    
    -- Key prefix for identification (first 8 chars of original key)
    -- Example: "sk_live_" or "ak_test_"
    key_prefix text not null,
    
    -- Scopes/permissions (JSON array)
    -- Example: ["monitors:read", "monitors:write", "incidents:read"]
    scopes jsonb default '[]'::jsonb,
    
    -- Rate limiting
    rate_limit_per_hour int default 1000,
    
    -- Key status
    is_active boolean default true,
    
    -- Expiration (optional)
    expires_at timestamptz,
    
    -- Usage tracking
    last_used_at timestamptz,
    total_requests bigint default 0,
    
    -- Timestamps
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

-- Index for fast lookups
create index if not exists idx_api_keys_key_hash on api_keys(key_hash);
create index if not exists idx_api_keys_user_id on api_keys(user_id);
create index if not exists idx_api_keys_is_active on api_keys(is_active);

-- Users table to store additional user metadata (optional)
-- This is separate from Clerk - stores local user data
create table if not exists users (
    -- Primary identifier (from Clerk user_id or generated for API customers)
    user_id text primary key,
    
    -- User type: 'clerk' or 'api_customer'
    user_type text not null check (user_type in ('clerk', 'api_customer')),
    
    -- Email (from Clerk or provided during API key creation)
    email text,
    
    -- Display name
    name text,
    
    -- Clerk-specific data (only for clerk users)
    clerk_user_id text unique,
    
    -- API customer specific data
    company_name text,
    
    -- Account status
    is_active boolean default true,
    
    -- Timestamps
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists idx_users_clerk_user_id on users(clerk_user_id);
create index if not exists idx_users_user_type on users(user_type);

-- Update existing monitors table to ensure user_id is text
-- (You mentioned you already changed this, but including for completeness)
alter table monitors alter column user_id type text;

-- Function to update updated_at timestamp
create or replace function update_updated_at_column()
returns trigger as $$
begin
    new.updated_at = now();
    return new;
end;
$$ language plpgsql;

-- Triggers for updated_at
drop trigger if exists update_api_keys_updated_at on api_keys;
create trigger update_api_keys_updated_at
    before update on api_keys
    for each row
    execute function update_updated_at_column();

drop trigger if exists update_users_updated_at on users;
create trigger update_users_updated_at
    before update on users
    for each row
    execute function update_updated_at_column();