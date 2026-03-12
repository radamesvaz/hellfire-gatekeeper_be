ALTER TABLE orders_history
    DROP CONSTRAINT IF EXISTS fk_orders_history_tenant;
DROP INDEX IF EXISTS idx_orders_history_tenant_id;
ALTER TABLE orders_history
    DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE order_items
    DROP CONSTRAINT IF EXISTS fk_order_items_tenant;
DROP INDEX IF EXISTS idx_order_items_tenant_id;
ALTER TABLE order_items
    DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS fk_orders_tenant;
DROP INDEX IF EXISTS idx_orders_tenant_id_status_created;
ALTER TABLE orders
    DROP COLUMN IF EXISTS tenant_id;

