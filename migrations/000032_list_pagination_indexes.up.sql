-- Support tenant-scoped list queries: products by id_product DESC, order_items by tenant + id_order.

CREATE INDEX idx_products_tenant_id_id_product_desc
    ON products (tenant_id, id_product DESC);

CREATE INDEX idx_order_items_tenant_id_id_order
    ON order_items (tenant_id, id_order);
