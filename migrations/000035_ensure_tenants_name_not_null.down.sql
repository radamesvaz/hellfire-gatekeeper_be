-- Revert NOT NULL on tenants.name (may allow NULLs again).
ALTER TABLE tenants
    ALTER COLUMN name DROP NOT NULL;
