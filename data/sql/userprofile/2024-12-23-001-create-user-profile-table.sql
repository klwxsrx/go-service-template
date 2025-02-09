create table if not exists user_profile (
    user_id    uuid primary key,
    first_name text        not null,
    last_name  text        not null,
    created_at timestamptz not null default current_timestamp,
    updated_at timestamptz not null default current_timestamp
)