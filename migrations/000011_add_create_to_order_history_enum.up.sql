-- Add 'create' to history_action enum
DO $$ BEGIN
    ALTER TYPE history_action ADD VALUE IF NOT EXISTS 'create';
EXCEPTION WHEN duplicate_object THEN null;
END $$;
