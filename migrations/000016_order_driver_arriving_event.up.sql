DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_enum
        WHERE enumlabel = 'driver.arriving'
          AND enumtypid = 'order_event_type'::regtype
    ) THEN
        ALTER TYPE order_event_type ADD VALUE 'driver.arriving' AFTER 'driver.location';
    END IF;
END $$;
