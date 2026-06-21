CREATE TABLE transport_request_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    protocol VARCHAR(16) NOT NULL,
    event_type TEXT NOT NULL,
    request_id TEXT,
    method VARCHAR(16),
    route TEXT,
    path TEXT NOT NULL,
    raw_query TEXT NOT NULL DEFAULT '',
    status_code INTEGER,
    duration_ms BIGINT,
    client_ip TEXT,
    user_agent TEXT,
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_role VARCHAR(64),
    error_message TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT transport_request_logs_protocol_check CHECK (protocol IN ('http', 'ws')),
    CONSTRAINT transport_request_logs_duration_non_negative CHECK (duration_ms IS NULL OR duration_ms >= 0),
    CONSTRAINT transport_request_logs_status_code_positive CHECK (status_code IS NULL OR status_code >= 100)
);

CREATE INDEX idx_transport_request_logs_protocol_created_at
ON transport_request_logs (protocol, created_at DESC);

CREATE INDEX idx_transport_request_logs_actor_user_created_at
ON transport_request_logs (actor_user_id, created_at DESC)
WHERE actor_user_id IS NOT NULL;

CREATE INDEX idx_transport_request_logs_request_id
ON transport_request_logs (request_id)
WHERE request_id IS NOT NULL;

CREATE INDEX idx_transport_request_logs_path_created_at
ON transport_request_logs (path, created_at DESC);
