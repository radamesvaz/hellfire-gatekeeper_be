-- Revert tenant_id changes on products_history
ALTER TABLE products_history
    DROP CONSTRAINT IF EXISTS fk_products_history_tenant;

DROP INDEX IF EXISTS idx_products_history_tenant_id;

ALTER TABLE products_history
    DROP COLUMN IF EXISTS tenant_id;

-- Revert tenant_id changes on products
ALTER TABLE products
    DROP CONSTRAINT IF EXISTS fk_products_tenant;

DROP INDEX IF EXISTS idx_products_tenant_id_status;

ALTER TABLE products
    DROP COLUMN IF EXISTS tenant_id;

