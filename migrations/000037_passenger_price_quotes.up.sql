CREATE TABLE passenger_price_quotes (
    id UUID PRIMARY KEY,
    passenger_id UUID NOT NULL REFERENCES passengers(id),
    city_id UUID NOT NULL,
    car_class_id UUID NOT NULL REFERENCES car_classes(id),
    fare_mode TEXT NOT NULL CHECK (fare_mode IN ('fixed_quote', 'metered')),
    pickup_latitude DOUBLE PRECISION NOT NULL,
    pickup_longitude DOUBLE PRECISION NOT NULL,
    destination_latitude DOUBLE PRECISION NOT NULL,
    destination_longitude DOUBLE PRECISION NOT NULL,
    pricing_snapshot JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_order_id UUID REFERENCES orders(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX passenger_price_quotes_passenger_expiry_idx
    ON passenger_price_quotes (passenger_id, expires_at DESC);
