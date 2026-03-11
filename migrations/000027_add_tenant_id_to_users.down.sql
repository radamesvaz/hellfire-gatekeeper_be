-- Revert tenant_id changes on users

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS ux_users_tenant_email;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS fk_users_tenant;

ALTER TABLE users
    DROP COLUMN IF EXISTS tenant_id;

-- Restore global uniqueness on email
ALTER TABLE users
    ADD CONSTRAINT users_email_key UNIQUE (email);

