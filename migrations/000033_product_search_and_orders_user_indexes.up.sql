-- List products: prefix filter uses lower(name) LIKE ... (see ListProductsPage).
CREATE INDEX idx_products_tenant_lower_name_prefix
    ON products (tenant_id, lower(name) varchar_pattern_ops);

-- List orders: optional filter id_user + sort by created_on, id_order.
CREATE INDEX idx_orders_tenant_id_user_created_id
    ON orders (tenant_id, id_user, created_on, id_order);
