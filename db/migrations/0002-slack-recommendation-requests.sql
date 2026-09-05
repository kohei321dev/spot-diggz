-- Short-lived Slack recommendation request state for retry deduplication.
CREATE TABLE IF NOT EXISTS slack_recommendation_requests (
    request_id TEXT PRIMARY KEY,
    source_event_key TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN (
        'generating',
        'delivered',
        'failed'
    )),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS slack_recommendation_requests_expires_at_idx
    ON slack_recommendation_requests (expires_at);
