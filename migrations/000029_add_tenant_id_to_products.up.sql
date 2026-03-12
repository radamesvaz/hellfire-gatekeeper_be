-- Add tenant_id to products and products_history for multi-tenant support

-- products: add tenant_id, backfill, enforce NOT NULL, FK and index
ALTER TABLE products
    ADD COLUMN tenant_id BIGINT;

UPDATE products
SET tenant_id = 1
WHERE tenant_id IS NULL;

ALTER TABLE products
    ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE products
    ADD CONSTRAINT fk_products_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);

CREATE INDEX idx_products_tenant_id_status
    ON products (tenant_id, status);

-- products_history: tenant_id follows the parent product
ALTER TABLE products_history
    ADD COLUMN tenant_id BIGINT;

UPDATE products_history ph
SET tenant_id = p.tenant_id
FROM products p
WHERE ph.id_product = p.id_product;

ALTER TABLE products_history
    ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE products_history
    ADD CONSTRAINT fk_products_history_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);

CREATE INDEX idx_products_history_tenant_id
    ON products_history (tenant_id, id_product);

