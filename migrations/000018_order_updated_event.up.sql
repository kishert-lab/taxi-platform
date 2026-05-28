DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_enum
        WHERE enumlabel = 'order.updated'
          AND enumtypid = 'order_event_type'::regtype
    ) THEN
        ALTER TYPE order_event_type ADD VALUE 'order.updated' AFTER 'order.rejected';
    END IF;
END $$;
