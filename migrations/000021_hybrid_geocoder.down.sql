DROP INDEX IF EXISTS local_geo_point_confirmations_point_idx;
DROP TABLE IF EXISTS local_geo_point_confirmations;
DROP TRIGGER IF EXISTS local_geo_points_set_updated_at ON local_geo_points;
DROP INDEX IF EXISTS local_geo_points_location_gist_idx;
DROP INDEX IF EXISTS local_geo_points_normalized_name_idx;
DROP INDEX IF EXISTS local_geo_points_city_trust_idx;
DROP TABLE IF EXISTS local_geo_points;
DROP TRIGGER IF EXISTS geocoder_external_cache_set_updated_at ON geocoder_external_cache;
DROP INDEX IF EXISTS geocoder_external_cache_expires_at_idx;
DROP INDEX IF EXISTS geocoder_external_cache_provider_query_city_idx;
DROP TABLE IF EXISTS geocoder_external_cache;

