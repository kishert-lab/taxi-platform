CREATE UNIQUE INDEX idx_order_route_points_order_recorded_location_unique
ON order_route_points (
    order_id,
    recorded_at,
    ((ST_Y(location::geometry))),
    ((ST_X(location::geometry)))
);
