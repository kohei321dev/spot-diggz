-- Neon/PostgreSQL schema for correction reports.
-- Apply this migration before setting DATABASE_URL in Vercel Production.
CREATE TABLE IF NOT EXISTS correction_reports (
    report_id TEXT PRIMARY KEY,
    facility_id TEXT NOT NULL,
    category TEXT NOT NULL,
    details TEXT NOT NULL,
    evidence_url TEXT,
    contact TEXT,
    contact_consent BOOLEAN NOT NULL DEFAULT FALSE,
    received_at TIMESTAMPTZ NOT NULL,
    delete_after TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS correction_reports_delete_after_idx
    ON correction_reports (delete_after);
