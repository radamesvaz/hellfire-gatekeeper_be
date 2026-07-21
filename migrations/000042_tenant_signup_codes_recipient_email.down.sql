DROP INDEX IF EXISTS idx_tenant_signup_codes_recipient_email;

ALTER TABLE tenant_signup_codes
    DROP COLUMN IF EXISTS recipient_email;
