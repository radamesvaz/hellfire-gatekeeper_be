-- subscription_status: VARCHAR -> PostgreSQL ENUM (active, pending, canceled)

DO $$ BEGIN
    CREATE TYPE subscription_status AS ENUM ('active', 'pending', 'canceled');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

ALTER TABLE tenants
    ALTER COLUMN subscription_status DROP DEFAULT;

ALTER TABLE tenants
    ALTER COLUMN subscription_status TYPE subscription_status
    USING (subscription_status::text::subscription_status);

ALTER TABLE tenants
    ALTER COLUMN subscription_status SET DEFAULT 'active';
