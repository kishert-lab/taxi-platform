CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE user_role AS ENUM ('passenger', 'driver', 'admin', 'dispatcher');
CREATE TYPE driver_status AS ENUM ('offline', 'online', 'busy', 'paused', 'blocked');
CREATE TYPE order_status AS ENUM (
    'created',
    'searching',
    'driver_assigned',
    'driver_arriving',
    'driver_waiting',
    'in_progress',
    'completed',
    'cancelled',
    'failed'
);
CREATE TYPE payment_method AS ENUM ('cash', 'card', 'corporate');
CREATE TYPE order_event_type AS ENUM (
    'order.created',
    'order.searching',
    'order.offer',
    'order.accepted',
    'order.rejected',
    'order.cancelled',
    'driver.location',
    'driver.arrived',
    'trip.started',
    'trip.completed',
    'trip.failed'
);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE cities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    region TEXT NOT NULL,
    country_code CHAR(2) NOT NULL DEFAULT 'RU',
    timezone TEXT NOT NULL DEFAULT 'Asia/Yekaterinburg',
    center geography(Point, 4326) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT cities_name_region_unique UNIQUE (name, region, country_code)
);

CREATE TRIGGER cities_set_updated_at
BEFORE UPDATE ON cities
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE zones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    city_id UUID NOT NULL REFERENCES cities(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    polygon geography(Polygon, 4326) NOT NULL,
    priority INTEGER NOT NULL DEFAULT 100,
    is_active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT zones_city_name_unique UNIQUE (city_id, name)
);

CREATE TRIGGER zones_set_updated_at
BEFORE UPDATE ON zones
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone TEXT NOT NULL,
    role user_role NOT NULL,
    first_name TEXT,
    last_name TEXT,
    password_hash TEXT,
    is_phone_confirmed BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT users_phone_role_unique UNIQUE (phone, role),
    CONSTRAINT users_phone_not_blank CHECK (length(trim(phone)) > 0)
);

CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    user_agent TEXT,
    ip_address INET,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE drivers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE RESTRICT,
    city_id UUID NOT NULL REFERENCES cities(id) ON DELETE RESTRICT,
    status driver_status NOT NULL DEFAULT 'offline',
    rating NUMERIC(3,2) NOT NULL DEFAULT 5.00,
    completed_orders_count INTEGER NOT NULL DEFAULT 0,
    license_number TEXT,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    blocked_reason TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT drivers_rating_range CHECK (rating >= 0 AND rating <= 5)
);

CREATE TRIGGER drivers_set_updated_at
BEFORE UPDATE ON drivers
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE cars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    brand TEXT NOT NULL,
    model TEXT NOT NULL,
    color TEXT NOT NULL,
    plate_number TEXT NOT NULL,
    year INTEGER,
    seats INTEGER NOT NULL DEFAULT 4,
    is_active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT cars_plate_number_unique UNIQUE (plate_number),
    CONSTRAINT cars_seats_positive CHECK (seats > 0)
);

CREATE TRIGGER cars_set_updated_at
BEFORE UPDATE ON cars
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE driver_locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL UNIQUE REFERENCES drivers(id) ON DELETE CASCADE,
    city_id UUID NOT NULL REFERENCES cities(id) ON DELETE RESTRICT,
    location geography(Point, 4326) NOT NULL,
    heading SMALLINT,
    speed_mps NUMERIC(6,2),
    accuracy_meters NUMERIC(8,2),
    recorded_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT driver_locations_heading_range CHECK (heading IS NULL OR (heading >= 0 AND heading <= 359)),
    CONSTRAINT driver_locations_accuracy_positive CHECK (accuracy_meters IS NULL OR accuracy_meters >= 0)
);

CREATE TRIGGER driver_locations_set_updated_at
BEFORE UPDATE ON driver_locations
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE tariffs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    city_id UUID NOT NULL REFERENCES cities(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    base_price NUMERIC(12,2) NOT NULL,
    price_per_km NUMERIC(12,2) NOT NULL,
    price_per_minute NUMERIC(12,2) NOT NULL,
    minimum_price NUMERIC(12,2) NOT NULL,
    free_waiting_minutes INTEGER NOT NULL DEFAULT 3,
    is_active BOOLEAN NOT NULL DEFAULT true,
    rules JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT tariffs_city_name_unique UNIQUE (city_id, name),
    CONSTRAINT tariffs_non_negative_prices CHECK (
        base_price >= 0 AND price_per_km >= 0 AND price_per_minute >= 0 AND minimum_price >= 0
    )
);

CREATE TRIGGER tariffs_set_updated_at
BEFORE UPDATE ON tariffs
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    passenger_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    driver_id UUID REFERENCES drivers(id) ON DELETE RESTRICT,
    city_id UUID NOT NULL REFERENCES cities(id) ON DELETE RESTRICT,
    tariff_id UUID REFERENCES tariffs(id) ON DELETE SET NULL,
    status order_status NOT NULL DEFAULT 'created',
    pickup_address TEXT NOT NULL,
    pickup_location geography(Point, 4326) NOT NULL,
    destination_address TEXT,
    destination_location geography(Point, 4326),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    accepted_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancellation_reason TEXT,
    estimated_price NUMERIC(12,2),
    final_price NUMERIC(12,2),
    payment_method payment_method NOT NULL DEFAULT 'cash',
    passenger_comment TEXT,
    dispatch_attempt INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT orders_price_non_negative CHECK (
        (estimated_price IS NULL OR estimated_price >= 0) AND (final_price IS NULL OR final_price >= 0)
    )
);

CREATE TRIGGER orders_set_updated_at
BEFORE UPDATE ON orders
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE order_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_driver_id UUID REFERENCES drivers(id) ON DELETE SET NULL,
    event_type order_event_type NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cities_active ON cities (is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_cities_center_gist ON cities USING GIST (center);

CREATE INDEX idx_zones_city_active ON zones (city_id, is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_zones_polygon_gist ON zones USING GIST (polygon);

CREATE INDEX idx_users_phone ON users (phone) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role_active ON users (role, is_active) WHERE deleted_at IS NULL;

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens (expires_at);

CREATE INDEX idx_drivers_city_status ON drivers (city_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_drivers_user_id ON drivers (user_id);

CREATE INDEX idx_cars_driver_active ON cars (driver_id, is_active) WHERE deleted_at IS NULL;

CREATE INDEX idx_driver_locations_city_recorded_at ON driver_locations (city_id, recorded_at DESC);
CREATE INDEX idx_driver_locations_location_gist ON driver_locations USING GIST (location);

CREATE INDEX idx_tariffs_city_active ON tariffs (city_id, is_active) WHERE deleted_at IS NULL;

CREATE INDEX idx_orders_passenger_created_at ON orders (passenger_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_driver_created_at ON orders (driver_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_city_status ON orders (city_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_pickup_location_gist ON orders USING GIST (pickup_location);
CREATE INDEX idx_orders_destination_location_gist ON orders USING GIST (destination_location);
CREATE UNIQUE INDEX idx_orders_one_active_per_passenger
ON orders (passenger_id)
WHERE status IN ('created', 'searching', 'driver_assigned', 'driver_arriving', 'driver_waiting', 'in_progress')
  AND deleted_at IS NULL;

CREATE INDEX idx_order_events_order_created_at ON order_events (order_id, created_at);
CREATE INDEX idx_order_events_type_created_at ON order_events (event_type, created_at DESC);
