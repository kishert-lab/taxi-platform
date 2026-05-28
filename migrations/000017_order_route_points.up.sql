CREATE TABLE order_route_points (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE RESTRICT,
    location geography(Point, 4326) NOT NULL,
    heading SMALLINT,
    speed_mps NUMERIC(8, 2),
    accuracy_meters NUMERIC(8, 2),
    recorded_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_order_route_points_order_recorded_at
ON order_route_points (order_id, recorded_at);

CREATE INDEX idx_order_route_points_driver_recorded_at
ON order_route_points (driver_id, recorded_at DESC);

CREATE INDEX idx_order_route_points_location_gist
ON order_route_points USING GIST (location);
