CREATE TABLE IF NOT EXISTS events (
    id          BIGSERIAL PRIMARY KEY,
    event_name  VARCHAR(100) NOT NULL,
    channel     VARCHAR(50)  NOT NULL,
    campaign_id VARCHAR(100),
    user_id     VARCHAR(100) NOT NULL,
    event_time  TIMESTAMPTZ  NOT NULL,
    tags        TEXT[]       NOT NULL DEFAULT '{}',
    metadata    JSONB        NOT NULL DEFAULT '{}'::jsonb,
    dedupe_key  TEXT         NOT NULL
    );

-- Idempotency için unique key
CREATE UNIQUE INDEX IF NOT EXISTS ux_events_dedupe
    ON events (dedupe_key);

-- Metrics sorguları için index'ler
CREATE INDEX IF NOT EXISTS idx_events_eventname_time
    ON events (event_name, event_time);

CREATE INDEX IF NOT EXISTS idx_events_eventname_time_channel
    ON events (event_name, event_time, channel);

CREATE INDEX IF NOT EXISTS idx_events_eventname_time_user
    ON events (event_name, event_time, user_id);
