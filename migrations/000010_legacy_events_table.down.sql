-- Restore the legacy table name (see the up migration).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema = 'public' AND table_name = 'events_legacy')
    THEN
        ALTER TABLE public.events_legacy RENAME TO events;
        RAISE NOTICE 'restored public.events_legacy -> public.events';
    END IF;
END $$;
