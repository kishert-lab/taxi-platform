DELETE FROM geocoder_external_cache
WHERE provider = 'dadata';

ALTER TABLE geocoder_external_cache
    DROP CONSTRAINT IF EXISTS geocoder_external_cache_provider_check;

ALTER TABLE geocoder_external_cache
    ADD CONSTRAINT geocoder_external_cache_provider_check
    CHECK (provider IN ('yandex'));
