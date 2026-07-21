-- Add recipient email for tenant signup code delivery (Brevo).
ALTER TABLE tenant_signup_codes
    ADD COLUMN recipient_email VARCHAR(320) NULL;

CREATE INDEX idx_tenant_signup_codes_recipient_email
    ON tenant_signup_codes (recipient_email);
