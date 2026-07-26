DELETE FROM user_document_acceptance
WHERE document_id IN (
    SELECT id
    FROM legal_documents
    WHERE document_type = 'privacy_policy'
      AND version = '2.1'
      AND language = 'ru'
);

DELETE FROM legal_documents
WHERE document_type = 'privacy_policy'
  AND version = '2.1'
  AND language = 'ru';

UPDATE legal_documents
SET is_active = true
WHERE document_type = 'privacy_policy'
  AND version = '2.0'
  AND language = 'ru';
