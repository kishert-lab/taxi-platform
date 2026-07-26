DROP INDEX IF EXISTS idx_passenger_ratings_driver_created_at;
DROP INDEX IF EXISTS idx_passenger_ratings_passenger_created_at;
DROP TABLE IF EXISTS passenger_ratings;

ALTER TABLE drivers DROP CONSTRAINT IF EXISTS drivers_ratings_count_non_negative;
ALTER TABLE drivers DROP COLUMN IF EXISTS ratings_count;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_ratings_count_non_negative;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_rating_range;
ALTER TABLE users DROP COLUMN IF EXISTS ratings_count;
ALTER TABLE users DROP COLUMN IF EXISTS rating;
ALTER TABLE users DROP COLUMN IF EXISTS profile_photo_url;

