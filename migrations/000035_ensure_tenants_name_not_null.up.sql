-- Defensive alignment with 000026: tenants.name must be NOT NULL and non-empty for app invariants.
UPDATE tenants
SET name = slug
WHERE name IS NULL OR btrim(name) = '';

ALTER TABLE tenants
    ALTER COLUMN name SET NOT NULL;
