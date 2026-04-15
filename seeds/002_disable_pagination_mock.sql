-- Disable/remove pagination demo seed data.
-- Safe to run multiple times (idempotent deletes by deterministic patterns).

BEGIN;

-- 1) Remove order items for seeded pagination orders.
DELETE FROM order_items oi
USING orders o
WHERE oi.id_order = o.id_order
  AND o.tenant_id IN (1, 2, 3)
  AND (
    o.note LIKE 'mock_seed_pagination_order_t1_%'
    OR o.note LIKE 'mock_seed_pagination_order_t2_%'
    OR o.note LIKE 'mock_seed_pagination_order_t3_%'
  );

-- 2) Remove seeded pagination orders.
DELETE FROM orders o
WHERE o.tenant_id IN (1, 2, 3)
  AND (
    o.note LIKE 'mock_seed_pagination_order_t1_%'
    OR o.note LIKE 'mock_seed_pagination_order_t2_%'
    OR o.note LIKE 'mock_seed_pagination_order_t3_%'
  );

-- 3) Remove product history for seeded pagination products.
DELETE FROM products_history ph
USING products p
WHERE ph.id_product = p.id_product
  AND ph.tenant_id = p.tenant_id
  AND p.tenant_id IN (1, 2, 3)
  AND p.name LIKE 'seed_pagination_t%';

-- 4) Remove seeded pagination products.
DELETE FROM products p
WHERE p.tenant_id IN (1, 2, 3)
  AND p.name LIKE 'seed_pagination_t%';

COMMIT;
