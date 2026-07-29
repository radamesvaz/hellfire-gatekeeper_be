-- Storefront WhatsApp contact for checkout invoice deep links (tenant-level, not admin user phone).
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS whatsapp_phone VARCHAR(20);
