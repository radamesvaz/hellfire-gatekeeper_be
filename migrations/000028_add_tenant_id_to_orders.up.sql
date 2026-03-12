-- Add tenant_id to orders, order_items and orders_history

ALTER TABLE orders
    ADD COLUMN tenant_id BIGINT;

UPDATE orders
SET tenant_id = 1
WHERE tenant_id IS NULL;

ALTER TABLE orders
    ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE orders
    ADD CONSTRAINT fk_orders_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id);

CREATE INDEX idx_orders_tenant_id_status_created
    ON orders (tenant_id, status, created_on);

-- order_items: tenant_id follows the parent order
ALTER TABLE order_items
    ADD COLUMN tenant_id BIGINT;

UPDATE order_items oi
SET tenant_id = o.tenant_id
FROM orders o
WHERE oi.id_order = o.id_order;

ALTER TABLE order_items
    ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE order_items
    ADD CONSTRAINT fk_order_items_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id);

CREATE INDEX idx_order_items_tenant_id
    ON order_items (tenant_id);

-- orders_history: tenant_id follows the parent order
ALTER TABLE orders_history
    ADD COLUMN tenant_id BIGINT;

UPDATE orders_history oh
SET tenant_id = o.tenant_id
FROM orders o
WHERE oh.id_order = o.id_order;

ALTER TABLE orders_history
    ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE orders_history
    ADD CONSTRAINT fk_orders_history_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id);

CREATE INDEX idx_orders_history_tenant_id
    ON orders_history (tenant_id, id_order);

