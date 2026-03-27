-- Logo display size is enforced by frontend layout + backend validation on upload; no persisted dimensions.
ALTER TABLE tenants
    DROP COLUMN IF EXISTS logo_width,
    DROP COLUMN IF EXISTS logo_height;
