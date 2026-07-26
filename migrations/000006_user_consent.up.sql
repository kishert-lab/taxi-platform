ALTER TABLE users
    ADD COLUMN personal_data_consent BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN personal_data_consent_at TIMESTAMPTZ,
    ADD COLUMN privacy_policy_version VARCHAR(50),
    ADD COLUMN terms_accepted BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN terms_accepted_at TIMESTAMPTZ,
    ADD COLUMN terms_version VARCHAR(50),
    ADD COLUMN consent_ip VARCHAR(64),
    ADD COLUMN consent_user_agent TEXT;

CREATE TABLE user_consent_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    event_type TEXT NOT NULL,
    document_type TEXT NOT NULL,
    document_version VARCHAR(50) NOT NULL,
    ip VARCHAR(64),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_consent_events_event_type_not_blank CHECK (length(trim(event_type)) > 0),
    CONSTRAINT user_consent_events_document_type_not_blank CHECK (length(trim(document_type)) > 0),
    CONSTRAINT user_consent_events_document_version_not_blank CHECK (length(trim(document_version)) > 0)
);

CREATE INDEX idx_user_consent_events_user_created_at
ON user_consent_events (user_id, created_at DESC);

CREATE INDEX idx_user_consent_events_document
ON user_consent_events (document_type, document_version);
