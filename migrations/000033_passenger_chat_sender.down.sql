DROP INDEX IF EXISTS idx_chat_messages_sender_passenger_id;

ALTER TABLE chat_messages
    DROP CONSTRAINT IF EXISTS chat_messages_sender_actor_check;

ALTER TABLE chat_messages
    DROP CONSTRAINT IF EXISTS chat_messages_sender_role_check;

ALTER TABLE chat_messages
    ADD CONSTRAINT chat_messages_sender_role_check CHECK (sender_role IN ('passenger', 'driver', 'taxi_park', 'dispatcher', 'admin'));

ALTER TABLE chat_messages
    DROP COLUMN IF EXISTS sender_passenger_id;

ALTER TABLE chat_messages
    ALTER COLUMN sender_user_id SET NOT NULL;
