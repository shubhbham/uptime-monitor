-- Complete Database Schema for Uptime Monitor API
-- Run this file on a fresh database

-- Enable required extensions
create extension if not exists "pgcrypto";

-- ============================================================================
-- USERS TABLE
-- ============================================================================
create table users (
    user_id text primary key,
    user_type text not null check (user_type in ('clerk', 'api_customer')),
    email text,
    name text,
    clerk_user_id text unique,
    company_name text,
    is_active boolean not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

-- Users indexes
create index idx_users_clerk_user_id on users(clerk_user_id);
create index idx_users_user_type on users(user_type);
create index idx_users_is_active on users(is_active);
create index idx_users_email on users(email);

-- ============================================================================
-- MONITORS TABLE
-- ============================================================================
create table monitors (
    id uuid primary key default gen_random_uuid(),
    user_id text not null,
    name text not null,
    url text not null,
    method text not null default 'GET',
    expected_status int not null default 200,
    interval_seconds int not null default 60,
    timeout_seconds int not null default 10,
    is_active boolean not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    
    -- Foreign key with CASCADE delete
    constraint fk_monitors_user_id 
        foreign key (user_id) 
        references users(user_id) 
        on delete cascade,
    
    -- Unique monitor name per user
    constraint uq_monitors_user_name 
        unique (user_id, name)
);

-- Monitors indexes
create index idx_monitors_user_id on monitors(user_id);
create index idx_monitors_is_active on monitors(is_active);
create index idx_monitors_user_id_is_active on monitors(user_id, is_active);

-- ============================================================================
-- MONITOR CHECKS TABLE
-- ============================================================================
create table monitor_checks (
    id bigserial primary key,
    monitor_id uuid not null,
    status_code int,
    response_time_ms int,
    is_up boolean not null,
    error_message text,
    checked_at timestamptz not null default now(),
    
    -- Foreign key with CASCADE delete
    constraint fk_monitor_checks_monitor_id 
        foreign key (monitor_id) 
        references monitors(id) 
        on delete cascade
);

-- Monitor checks indexes
create index idx_monitor_checks_monitor_id on monitor_checks(monitor_id);
create index idx_monitor_checks_checked_at on monitor_checks(checked_at desc);
create index idx_monitor_checks_monitor_id_checked_at on monitor_checks(monitor_id, checked_at desc);

-- ============================================================================
-- INCIDENTS TABLE
-- ============================================================================
create table incidents (
    id bigserial primary key,
    monitor_id uuid not null,
    started_at timestamptz not null,
    resolved_at timestamptz,
    cause text,
    is_resolved boolean not null default false,
    
    -- Foreign key with CASCADE delete
    constraint fk_incidents_monitor_id 
        foreign key (monitor_id) 
        references monitors(id) 
        on delete cascade
);

-- Incidents indexes
create index idx_incidents_monitor_id on incidents(monitor_id);
create index idx_incidents_started_at on incidents(started_at desc);
create index idx_incidents_monitor_id_is_resolved on incidents(monitor_id, is_resolved);

-- ============================================================================
-- MONITOR STATS TABLE
-- ============================================================================
create table monitor_stats (
    monitor_id uuid primary key,
    uptime_percentage numeric(5,2) not null default 100.00,
    avg_response_time_ms int,
    last_checked_at timestamptz,
    last_status boolean,
    
    -- Foreign key with CASCADE delete
    constraint fk_monitor_stats_monitor_id 
        foreign key (monitor_id) 
        references monitors(id) 
        on delete cascade
);

-- Monitor stats indexes
create index idx_monitor_stats_last_checked_at on monitor_stats(last_checked_at desc);

-- ============================================================================
-- API KEYS TABLE
-- ============================================================================
create table api_keys (
    id uuid primary key default gen_random_uuid(),
    user_id text not null,
    key_hash text not null,
    name text not null,
    key_prefix text not null,
    scopes jsonb not null default '[]'::jsonb,
    rate_limit_per_hour int not null default 1000,
    is_active boolean not null default true,
    expires_at timestamptz,
    last_used_at timestamptz,
    total_requests bigint not null default 0,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    
    -- Foreign key with CASCADE delete
    constraint fk_api_keys_user_id 
        foreign key (user_id) 
        references users(user_id) 
        on delete cascade,
    
    -- Unique key hash (prevents duplicate API keys)
    constraint uq_api_keys_key_hash 
        unique (key_hash),
    
    -- Unique key name per user
    constraint uq_api_keys_user_name 
        unique (user_id, name)
);

-- API keys indexes
create index idx_api_keys_key_hash on api_keys(key_hash) where is_active = true;
create index idx_api_keys_user_id on api_keys(user_id);
create index idx_api_keys_is_active on api_keys(is_active);
create index idx_api_keys_expires_at on api_keys(expires_at) where expires_at is not null;

-- ============================================================================
-- FUNCTIONS
-- ============================================================================

-- Function to update updated_at timestamp
create or replace function update_updated_at_column()
returns trigger as $$
begin
    new.updated_at = now();
    return new;
end;
$$ language plpgsql;

-- Function to disable user's monitors when user is deactivated
create or replace function disable_user_monitors()
returns trigger as $$
begin
    if new.is_active = false and old.is_active = true then
        update monitors 
        set is_active = false, updated_at = now()
        where user_id = new.user_id and is_active = true;
        
        raise notice 'Disabled % monitors for user %', 
            (select count(*) from monitors where user_id = new.user_id and is_active = false),
            new.user_id;
    end if;
    return new;
end;
$$ language plpgsql;

-- Function to disable user's API keys when user is deactivated
create or replace function disable_user_api_keys()
returns trigger as $$
begin
    if new.is_active = false and old.is_active = true then
        update api_keys 
        set is_active = false, updated_at = now()
        where user_id = new.user_id and is_active = true;
        
        raise notice 'Disabled % API keys for user %',
            (select count(*) from api_keys where user_id = new.user_id and is_active = false),
            new.user_id;
    end if;
    return new;
end;
$$ language plpgsql;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Trigger to update updated_at on api_keys
create trigger update_api_keys_updated_at
    before update on api_keys
    for each row
    execute function update_updated_at_column();

-- Trigger to update updated_at on users
create trigger update_users_updated_at
    before update on users
    for each row
    execute function update_updated_at_column();

-- Trigger to auto-disable monitors when user is deactivated
create trigger trigger_disable_user_monitors
    after update on users
    for each row
    when (new.is_active = false and old.is_active = true)
    execute function disable_user_monitors();

-- Trigger to auto-disable API keys when user is deactivated
create trigger trigger_disable_user_api_keys
    after update on users
    for each row
    when (new.is_active = false and old.is_active = true)
    execute function disable_user_api_keys();

-- ============================================================================
-- VIEWS
-- ============================================================================

-- View for user statistics
create or replace view user_stats as
select 
    u.user_id,
    u.email,
    u.name,
    u.user_type,
    u.is_active,
    count(distinct m.id) as total_monitors,
    count(distinct m.id) filter (where m.is_active) as active_monitors,
    count(distinct ak.id) as total_api_keys,
    count(distinct ak.id) filter (where ak.is_active) as active_api_keys,
    u.created_at,
    u.updated_at
from users u
left join monitors m on u.user_id = m.user_id
left join api_keys ak on u.user_id = ak.user_id
group by u.user_id;

-- ============================================================================
-- COMMENTS (Documentation)
-- ============================================================================

comment on table users is 'Stores user accounts (Clerk users and API customers)';
comment on table monitors is 'Stores monitored endpoints configuration';
comment on table monitor_checks is 'Stores individual monitor check results (time-series data)';
comment on table incidents is 'Stores downtime incidents (aggregated from checks)';
comment on table monitor_stats is 'Stores cached monitor statistics for fast retrieval';
comment on table api_keys is 'Stores hashed API keys for API-as-a-Service access';

comment on constraint fk_monitors_user_id on monitors is 'CASCADE DELETE: Deleting a user automatically deletes all their monitors';
comment on constraint fk_monitor_checks_monitor_id on monitor_checks is 'CASCADE DELETE: Deleting a monitor automatically deletes all its checks';
comment on constraint fk_incidents_monitor_id on incidents is 'CASCADE DELETE: Deleting a monitor automatically deletes all its incidents';
comment on constraint fk_monitor_stats_monitor_id on monitor_stats is 'CASCADE DELETE: Deleting a monitor automatically deletes its stats';
comment on constraint fk_api_keys_user_id on api_keys is 'CASCADE DELETE: Deleting a user automatically deletes all their API keys';

comment on constraint uq_monitors_user_name on monitors is 'Prevents duplicate monitor names within a single user account';
comment on constraint uq_api_keys_key_hash on api_keys is 'Ensures API key uniqueness across the entire system';
comment on constraint uq_api_keys_user_name on api_keys is 'Prevents duplicate API key names within a single user account';

-- ============================================================================
-- VERIFICATION
-- ============================================================================

-- Verify all tables created
do $$
declare
    table_count int;
begin
    select count(*) into table_count
    from information_schema.tables
    where table_schema = 'public'
    and table_name in ('users', 'monitors', 'monitor_checks', 'incidents', 'monitor_stats', 'api_keys');
    
    if table_count = 6 then
        raise notice 'SUCCESS: All 6 tables created';
    else
        raise warning 'WARNING: Expected 6 tables, found %', table_count;
    end if;
end $$;

-- Verify all foreign keys
do $$
declare
    fk_count int;
begin
    select count(*) into fk_count
    from information_schema.table_constraints
    where constraint_type = 'FOREIGN KEY'
    and table_schema = 'public';
    
    if fk_count >= 5 then
        raise notice 'SUCCESS: All foreign keys created (found %)', fk_count;
    else
        raise warning 'WARNING: Expected at least 5 foreign keys, found %', fk_count;
    end if;
end $$;

-- Verify all indexes
do $$
declare
    idx_count int;
begin
    select count(*) into idx_count
    from pg_indexes
    where schemaname = 'public';
    
    raise notice 'Created % indexes', idx_count;
end $$;

-- Display summary
select 
    'Schema creation completed!' as status,
    (select count(*) from information_schema.tables where table_schema = 'public') as tables,
    (select count(*) from information_schema.table_constraints where constraint_type = 'FOREIGN KEY' and table_schema = 'public') as foreign_keys,
    (select count(*) from information_schema.table_constraints where constraint_type = 'UNIQUE' and table_schema = 'public') as unique_constraints,
    (select count(*) from pg_indexes where schemaname = 'public') as indexes,
    (select count(*) from pg_trigger where tgname not like 'pg_%') as triggers,
    now() as created_at;