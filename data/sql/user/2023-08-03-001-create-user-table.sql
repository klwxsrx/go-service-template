create table if not exists "user" (
    id            uuid primary key,
    login         text        not null unique,
    password_hash text        not null,
    created_at    timestamptz not null default current_timestamp,
    updated_at    timestamptz not null default current_timestamp,
    deleted_at    timestamptz          default null
)