CREATE TABLE IF NOT EXISTS message_outbox (
    id      uuid PRIMARY KEY,
    topic   text,
    key     text,
    payload bytea
)