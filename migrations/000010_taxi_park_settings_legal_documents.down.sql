DROP INDEX IF EXISTS idx_user_document_acceptance_user_accepted_at;
DROP TABLE IF EXISTS user_document_acceptance;
DROP INDEX IF EXISTS idx_legal_documents_type_created_at;
DROP INDEX IF EXISTS idx_legal_documents_one_active;
DROP INDEX IF EXISTS idx_legal_documents_type_version_language;
DROP TABLE IF EXISTS legal_documents;
DROP TRIGGER IF EXISTS taxi_park_tariffs_set_updated_at ON taxi_park_tariffs;
DROP TABLE IF EXISTS taxi_park_tariffs;
DROP TRIGGER IF EXISTS taxi_park_settings_set_updated_at ON taxi_park_settings;
DROP TABLE IF EXISTS taxi_park_settings;

