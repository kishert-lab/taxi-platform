ALTER TABLE chat_messages
    ALTER COLUMN sender_user_id DROP NOT NULL;

ALTER TABLE chat_messages
    ADD COLUMN IF NOT EXISTS sender_passenger_id UUID;

ALTER TABLE chat_messages
    DROP CONSTRAINT IF EXISTS chat_messages_sender_passenger_id_fkey;

ALTER TABLE chat_messages
    ADD CONSTRAINT chat_messages_sender_passenger_id_fkey
    FOREIGN KEY (sender_passenger_id) REFERENCES passengers(id) ON DELETE RESTRICT;

ALTER TABLE chat_messages
    DROP CONSTRAINT IF EXISTS chat_messages_sender_role_check;

ALTER TABLE chat_messages
    ADD CONSTRAINT chat_messages_sender_role_check CHECK (sender_role IN ('passenger', 'driver', 'taxi_park', 'dispatcher', 'admin'));

ALTER TABLE chat_messages
    DROP CONSTRAINT IF EXISTS chat_messages_sender_actor_check;

ALTER TABLE chat_messages
    ADD CONSTRAINT chat_messages_sender_actor_check CHECK (
        (
            sender_role = 'passenger'
            AND sender_passenger_id IS NOT NULL
            AND sender_user_id IS NULL
        )
        OR (
            sender_role IN ('driver', 'taxi_park', 'dispatcher', 'admin')
            AND sender_user_id IS NOT NULL
            AND sender_passenger_id IS NULL
        )
    );

CREATE INDEX IF NOT EXISTS idx_chat_messages_sender_passenger_id
ON chat_messages (sender_passenger_id, created_at DESC)
WHERE deleted_at IS NULL;
