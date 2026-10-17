-- Explicitly handle the legacy `public.events` table.
--
-- Some environments (including the local dev database) carry a `public.events`
-- table that no migration creates; the canonical table is `blockchain_events`
-- (see 000001_init_schema.up.sql). Leaving it in place means two tables with
-- near-identical columns and no owner, which is how the schema/code split in
-- finding F04 stayed hidden.
--
-- The table is RENAMED rather than dropped so the change is reversible and no
-- data can be lost, whatever the environment holds.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema = 'public' AND table_name = 'events')
       AND NOT EXISTS (SELECT 1 FROM information_schema.tables
                       WHERE table_schema = 'public' AND table_name = 'events_legacy')
    THEN
        ALTER TABLE public.events RENAME TO events_legacy;
        RAISE NOTICE 'renamed legacy public.events -> public.events_legacy';
    END IF;
END $$;
