-- Remove the discontinued Lists feature and shorten transient Slack request retention.
DROP TABLE IF EXISTS saved_facilities;

ALTER TABLE slack_recommendation_requests
    DROP CONSTRAINT IF EXISTS slack_recommendation_requests_status_check;

ALTER TABLE slack_recommendation_requests
    ADD CONSTRAINT slack_recommendation_requests_status_check
    CHECK (status IN (
        'generating', 'delivered', 'failed',
        'awaiting_selection', 'saving', 'saved', 'rejected'
    ));

UPDATE slack_recommendation_requests
SET expires_at = LEAST(expires_at, updated_at + INTERVAL '1 hour');
