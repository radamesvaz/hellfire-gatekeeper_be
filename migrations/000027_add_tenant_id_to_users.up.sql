-- Add tenant_id to users and scope email uniqueness per tenant

ALTER TABLE users
    ADD COLUMN tenant_id BIGINT;

-- Backfill existing users with the default tenant (id = 1)
UPDATE users
SET tenant_id = 1
WHERE tenant_id IS NULL;

-- Enforce NOT NULL after backfill
ALTER TABLE users
    ALTER COLUMN tenant_id SET NOT NULL;

-- Foreign key to tenants
ALTER TABLE users
    ADD CONSTRAINT fk_users_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id);

-- Change email uniqueness from global to per-tenant
ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_email_key;

ALTER TABLE users
    ADD CONSTRAINT ux_users_tenant_email UNIQUE (tenant_id, email);

