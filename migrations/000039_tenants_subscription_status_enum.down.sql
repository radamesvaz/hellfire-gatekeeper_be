ALTER TABLE tenants
    ALTER COLUMN subscription_status DROP DEFAULT;

ALTER TABLE tenants
    ALTER COLUMN subscription_status TYPE VARCHAR(16)
    USING (subscription_status::text);

ALTER TABLE tenants
    ALTER COLUMN subscription_status SET DEFAULT 'active';

DROP TYPE IF EXISTS subscription_status;
