ALTER TABLE users
    ADD COLUMN IF NOT EXISTS profile_photo_url TEXT,
    ADD COLUMN IF NOT EXISTS rating NUMERIC(3,2) NOT NULL DEFAULT 5.00,
    ADD COLUMN IF NOT EXISTS ratings_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE users
    ADD CONSTRAINT users_rating_range
    CHECK (rating >= 0 AND rating <= 5);

ALTER TABLE users
    ADD CONSTRAINT users_ratings_count_non_negative
    CHECK (ratings_count >= 0);

ALTER TABLE drivers
    ADD COLUMN IF NOT EXISTS ratings_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE drivers
    ADD CONSTRAINT drivers_ratings_count_non_negative
    CHECK (ratings_count >= 0);

CREATE TABLE passenger_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    passenger_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE RESTRICT,
    score INTEGER NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT passenger_ratings_score_range CHECK (score BETWEEN 1 AND 5),
    CONSTRAINT passenger_ratings_order_driver_unique UNIQUE (order_id, driver_id)
);

CREATE INDEX idx_passenger_ratings_passenger_created_at
ON passenger_ratings (passenger_id, created_at DESC);

CREATE INDEX idx_passenger_ratings_driver_created_at
ON passenger_ratings (driver_id, created_at DESC);

