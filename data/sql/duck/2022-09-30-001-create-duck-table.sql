create table if not exists duck(
    id         uuid primary key,
    name       text        not null,
    is_active  boolean     not null,
    created_at timestamptz not null default current_timestamp,
    updated_at timestamptz not null default current_timestamp
)