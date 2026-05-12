DROP INDEX IF EXISTS idx_user_consent_events_document;
DROP INDEX IF EXISTS idx_user_consent_events_user_created_at;
DROP TABLE IF EXISTS user_consent_events;

ALTER TABLE users
    DROP COLUMN IF EXISTS consent_user_agent,
    DROP COLUMN IF EXISTS consent_ip,
    DROP COLUMN IF EXISTS terms_version,
    DROP COLUMN IF EXISTS terms_accepted_at,
    DROP COLUMN IF EXISTS terms_accepted,
    DROP COLUMN IF EXISTS privacy_policy_version,
    DROP COLUMN IF EXISTS personal_data_consent_at,
    DROP COLUMN IF EXISTS personal_data_consent;
