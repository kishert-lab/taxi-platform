CREATE TABLE chat_threads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_type TEXT NOT NULL,
    order_id UUID REFERENCES orders(id) ON DELETE CASCADE,
    taxi_park_id UUID REFERENCES taxi_parks(id) ON DELETE SET NULL,
    passenger_id UUID REFERENCES users(id) ON DELETE SET NULL,
    driver_id UUID REFERENCES drivers(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'open',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chat_threads_type_check CHECK (chat_type IN ('dispatcher_driver', 'driver_passenger', 'passenger_support')),
    CONSTRAINT chat_threads_status_check CHECK (status IN ('open', 'closed', 'archived')),
    CONSTRAINT chat_threads_order_required CHECK (
        (chat_type IN ('dispatcher_driver', 'driver_passenger') AND order_id IS NOT NULL)
        OR chat_type = 'passenger_support'
    )
);

CREATE TRIGGER chat_threads_set_updated_at
BEFORE UPDATE ON chat_threads
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE UNIQUE INDEX idx_chat_threads_order_type_unique
ON chat_threads (order_id, chat_type)
WHERE order_id IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX idx_chat_threads_passenger_support_unique
ON chat_threads (passenger_id, chat_type)
WHERE order_id IS NULL AND chat_type = 'passenger_support' AND deleted_at IS NULL;

CREATE INDEX idx_chat_threads_order_id
ON chat_threads (order_id)
WHERE deleted_at IS NULL;

CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id UUID NOT NULL REFERENCES chat_threads(id) ON DELETE CASCADE,
    order_id UUID REFERENCES orders(id) ON DELETE CASCADE,
    sender_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    sender_role TEXT NOT NULL,
    body TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    edited_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chat_messages_sender_role_check CHECK (sender_role IN ('passenger', 'driver', 'taxi_park', 'dispatcher', 'admin')),
    CONSTRAINT chat_messages_body_not_empty CHECK (length(trim(body)) > 0 AND length(body) <= 2000)
);

CREATE INDEX idx_chat_messages_thread_created_at
ON chat_messages (thread_id, created_at DESC)
WHERE deleted_at IS NULL;

CREATE INDEX idx_chat_messages_order_created_at
ON chat_messages (order_id, created_at DESC)
WHERE deleted_at IS NULL;
