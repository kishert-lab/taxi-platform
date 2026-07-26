CREATE TABLE order_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
    passenger_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE RESTRICT,
    score SMALLINT NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT order_ratings_score_range CHECK (score >= 1 AND score <= 5)
);

CREATE INDEX idx_order_ratings_driver_created_at
ON order_ratings (driver_id, created_at DESC);

CREATE INDEX idx_order_ratings_passenger_created_at
ON order_ratings (passenger_id, created_at DESC);
