CREATE TABLE geocoder_external_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL,
    normalized_query TEXT NOT NULL,
    city_id UUID REFERENCES cities(id) ON DELETE SET NULL,
    request_params JSONB NOT NULL DEFAULT '{}'::jsonb,
    response JSONB NOT NULL DEFAULT '{}'::jsonb,
    results JSONB NOT NULL DEFAULT '[]'::jsonb,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT geocoder_external_cache_provider_check CHECK (provider IN ('yandex')),
    CONSTRAINT geocoder_external_cache_query_not_blank CHECK (length(trim(normalized_query)) > 0)
);

CREATE UNIQUE INDEX geocoder_external_cache_provider_query_city_idx
ON geocoder_external_cache (provider, normalized_query, COALESCE(city_id, '00000000-0000-0000-0000-000000000000'::uuid));

CREATE INDEX geocoder_external_cache_expires_at_idx
ON geocoder_external_cache (expires_at);

CREATE TRIGGER geocoder_external_cache_set_updated_at
BEFORE UPDATE ON geocoder_external_cache
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE local_geo_points (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    city_id UUID NOT NULL REFERENCES cities(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    normalized_name TEXT NOT NULL,
    address TEXT NOT NULL,
    location geography(Point, 4326) NOT NULL,
    source TEXT NOT NULL DEFAULT 'user_confirmed',
    external_provider TEXT,
    external_place_id TEXT,
    confidence NUMERIC(5,4) NOT NULL DEFAULT 1.0000,
    trust_level TEXT NOT NULL DEFAULT 'confirmed',
    confirmation_count INTEGER NOT NULL DEFAULT 0,
    reject_count INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,
    rejected_by UUID REFERENCES users(id) ON DELETE SET NULL,
    rejected_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT local_geo_points_trust_level_check CHECK (trust_level IN ('confirmed', 'trusted', 'rejected')),
    CONSTRAINT local_geo_points_source_check CHECK (source IN ('user_confirmed', 'driver_confirmed', 'dispatcher_confirmed', 'admin')),
    CONSTRAINT local_geo_points_confirmation_count_check CHECK (confirmation_count >= 0),
    CONSTRAINT local_geo_points_reject_count_check CHECK (reject_count >= 0),
    CONSTRAINT local_geo_points_confidence_check CHECK (confidence >= 0 AND confidence <= 1)
);

CREATE INDEX local_geo_points_city_trust_idx
ON local_geo_points (city_id, trust_level)
WHERE deleted_at IS NULL;

CREATE INDEX local_geo_points_normalized_name_idx
ON local_geo_points (normalized_name)
WHERE deleted_at IS NULL;

CREATE INDEX local_geo_points_location_gist_idx
ON local_geo_points
USING GIST (location);

CREATE TRIGGER local_geo_points_set_updated_at
BEFORE UPDATE ON local_geo_points
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE local_geo_point_confirmations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    point_id UUID NOT NULL REFERENCES local_geo_points(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_role TEXT,
    action TEXT NOT NULL DEFAULT 'confirm',
    source TEXT NOT NULL DEFAULT 'user_confirmed',
    address TEXT NOT NULL,
    location geography(Point, 4326) NOT NULL,
    comment TEXT,
    ip TEXT,
    user_agent TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT local_geo_point_confirmations_action_check CHECK (action IN ('confirm', 'reject')),
    CONSTRAINT local_geo_point_confirmations_source_check CHECK (source IN ('user_confirmed', 'driver_confirmed', 'dispatcher_confirmed', 'admin'))
);

CREATE INDEX local_geo_point_confirmations_point_idx
ON local_geo_point_confirmations (point_id, created_at DESC);

