DROP INDEX IF EXISTS idx_user_consents_active_unique;
DROP INDEX IF EXISTS idx_user_consents_type_version;
DROP INDEX IF EXISTS idx_user_consents_user_accepted_at;
DROP TABLE IF EXISTS user_consents;

DELETE FROM user_document_acceptance
WHERE document_id IN (
    SELECT id FROM legal_documents
    WHERE (document_type IN ('license_agreement', 'personal_data_transfer', 'cookies_required', 'cookies_analytics', 'cookies_marketing', 'taxi_park_responsibility', 'driver_documents_processing', 'geo_data_processing') OR version IN ('2.0'))
      AND language = 'ru'
);

DELETE FROM legal_documents
WHERE (document_type IN ('license_agreement', 'personal_data_transfer', 'cookies_required', 'cookies_analytics', 'cookies_marketing', 'taxi_park_responsibility', 'driver_documents_processing', 'geo_data_processing') OR version IN ('2.0'))
  AND language = 'ru';

UPDATE legal_documents
SET is_active = true
WHERE document_type IN ('privacy_policy', 'terms_of_service', 'driver_agreement', 'taxi_park_agreement', 'consent_personal_data')
  AND version = '1.0'
  AND language = 'ru';

ALTER TABLE legal_documents
    DROP CONSTRAINT IF EXISTS legal_documents_type_check;

ALTER TABLE legal_documents
    ADD CONSTRAINT legal_documents_type_check
    CHECK (document_type IN ('privacy_policy', 'terms_of_service', 'driver_agreement', 'taxi_park_agreement', 'consent_personal_data'));
