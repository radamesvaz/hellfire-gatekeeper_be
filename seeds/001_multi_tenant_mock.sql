-- Local dev only: datos multi-tenant de prueba (idempotente).
-- Ejecutar tras `migrate up`. Ver seeds/README.md

-- bcrypt cost 10: adminpass
-- bcrypt cost 10: clientpass

UPDATE users
SET password_hash = '$2a$10$Ued/UmuZslCPE.c.iRdFEON7jicWeAlBBM3vjLrcktSg778XHtvQW'
WHERE id_role = (SELECT id_role FROM roles WHERE name = 'admin');

UPDATE users
SET password_hash = '$2a$10$VdS6bqypHg1a5ZJjKOovjuo4KRwMMegIJ11MZCDYLB/fufAY9OPh6'
WHERE id_role = (SELECT id_role FROM roles WHERE name = 'client');

INSERT INTO tenants (id, name, slug, is_active, ghost_order_timeout_minutes, subscription_status, plan_code)
VALUES
    (2, 'Panadería Norte', 'panaderia-norte', TRUE, 30, 'active', 'basic'),
    (3, 'Pastelería Centro', 'pasteleria-centro', TRUE, 30, 'active', 'basic')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (tenant_id, id_role, name, email, phone, password_hash, created_on)
VALUES
    (2, (SELECT id_role FROM roles WHERE name = 'admin'), 'Admin Norte', 'admin.norte@demo.local', '55-1000',
     '$2a$10$Ued/UmuZslCPE.c.iRdFEON7jicWeAlBBM3vjLrcktSg778XHtvQW', NOW()),
    (3, (SELECT id_role FROM roles WHERE name = 'admin'), 'Admin Centro', 'admin.centro@demo.local', '55-2000',
     '$2a$10$Ued/UmuZslCPE.c.iRdFEON7jicWeAlBBM3vjLrcktSg778XHtvQW', NOW())
ON CONFLICT (tenant_id, email) DO NOTHING;

INSERT INTO users (tenant_id, id_role, name, email, phone, password_hash, created_on)
SELECT t.id,
       (SELECT id_role FROM roles WHERE name = 'client'),
       'Cliente Compartido',
       'shared.client@demo.local',
       '55-9999',
       '$2a$10$VdS6bqypHg1a5ZJjKOovjuo4KRwMMegIJ11MZCDYLB/fufAY9OPh6',
       NOW()
FROM tenants t
WHERE t.slug IN ('default', 'panaderia-norte', 'pasteleria-centro')
ON CONFLICT (tenant_id, email) DO NOTHING;

INSERT INTO products (name, description, price, available, created_on, status, tenant_id, stock, image_urls, thumbnail_url)
SELECT 'Tarta Negra Norte', 'Mock tarta chocolate', 12.50, TRUE, NOW(), 'active', 2, 20, NULL, NULL
WHERE NOT EXISTS (SELECT 1 FROM products WHERE tenant_id = 2 AND name = 'Tarta Negra Norte');

INSERT INTO products (name, description, price, available, created_on, status, tenant_id, stock, image_urls, thumbnail_url)
SELECT 'Pan Integral Norte', 'Mock pan integral', 4.00, TRUE, NOW(), 'active', 2, 50, NULL, NULL
WHERE NOT EXISTS (SELECT 1 FROM products WHERE tenant_id = 2 AND name = 'Pan Integral Norte');

INSERT INTO products (name, description, price, available, created_on, status, tenant_id, stock, image_urls, thumbnail_url)
SELECT 'Milhojas Centro', 'Mock milhojas', 18.00, TRUE, NOW(), 'active', 3, 15, NULL, NULL
WHERE NOT EXISTS (SELECT 1 FROM products WHERE tenant_id = 3 AND name = 'Milhojas Centro');

INSERT INTO products (name, description, price, available, created_on, status, tenant_id, stock, image_urls, thumbnail_url)
SELECT 'Cupcake Centro', 'Mock cupcake', 3.75, TRUE, NOW(), 'active', 3, 40, NULL, NULL
WHERE NOT EXISTS (SELECT 1 FROM products WHERE tenant_id = 3 AND name = 'Cupcake Centro');

INSERT INTO products_history (
    tenant_id, id_product, name, description, price, available, stock, status, image_urls, thumbnail_url, modified_by, action
)
SELECT
    p.tenant_id,
    p.id_product,
    p.name,
    p.description,
    p.price,
    p.available,
    p.stock,
    p.status,
    p.image_urls,
    p.thumbnail_url,
    u.id_user,
    'create'::history_action
FROM products p
JOIN users u ON u.tenant_id = p.tenant_id AND u.id_role = (SELECT id_role FROM roles WHERE name = 'admin')
WHERE p.tenant_id IN (2, 3)
  AND NOT EXISTS (
      SELECT 1 FROM products_history ph
      WHERE ph.id_product = p.id_product AND ph.tenant_id = p.tenant_id AND ph.action = 'create'::history_action
  );

-- Pedidos tenant 2
WITH o1 AS (
    INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, delivery_direction, paid, expires_at)
    SELECT
        2,
        u.id_user,
        'delivered'::order_status,
        25.00,
        'mock_seed: t2 delivered',
        '2026-02-01 10:00:00+00'::timestamptz,
        '2026-02-05'::date,
        'https://maps.app.goo.gl/mock-norte',
        TRUE,
        NULL
    FROM users u
    WHERE u.tenant_id = 2 AND u.email = 'shared.client@demo.local'
      AND NOT EXISTS (SELECT 1 FROM orders WHERE tenant_id = 2 AND note = 'mock_seed: t2 delivered')
    RETURNING id_order, tenant_id
)
INSERT INTO order_items (tenant_id, id_order, id_product, product_name_snapshot, unit_price_snapshot, quantity)
SELECT o1.tenant_id, o1.id_order, p.id_product, p.name, p.price, 2
FROM o1
JOIN products p ON p.tenant_id = 2 AND p.name = 'Tarta Negra Norte';

WITH o2 AS (
    INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, delivery_direction, paid, expires_at)
    SELECT
        2,
        u.id_user,
        'pending'::order_status,
        8.00,
        'mock_seed: t2 pending future',
        '2026-03-30 12:00:00+00'::timestamptz,
        '2026-04-10'::date,
        'https://maps.app.goo.gl/mock-norte',
        FALSE,
        '2026-04-01 23:59:59+00'::timestamptz
    FROM users u
    WHERE u.tenant_id = 2 AND u.email = 'shared.client@demo.local'
      AND NOT EXISTS (SELECT 1 FROM orders WHERE tenant_id = 2 AND note = 'mock_seed: t2 pending future')
    RETURNING id_order, tenant_id
)
INSERT INTO order_items (tenant_id, id_order, id_product, product_name_snapshot, unit_price_snapshot, quantity)
SELECT o2.tenant_id, o2.id_order, p.id_product, p.name, p.price, 2
FROM o2
JOIN products p ON p.tenant_id = 2 AND p.name = 'Pan Integral Norte';

WITH o3 AS (
    INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, delivery_direction, paid, expires_at)
    SELECT
        2,
        u.id_user,
        'pending'::order_status,
        12.50,
        'mock_seed: t2 pending expired (cron)',
        '2026-03-20 08:00:00+00'::timestamptz,
        '2026-03-21'::date,
        'https://maps.app.goo.gl/mock-norte',
        FALSE,
        '2026-03-20 08:30:00+00'::timestamptz
    FROM users u
    WHERE u.tenant_id = 2 AND u.email = 'shared.client@demo.local'
      AND NOT EXISTS (SELECT 1 FROM orders WHERE tenant_id = 2 AND note = 'mock_seed: t2 pending expired (cron)')
    RETURNING id_order, tenant_id
)
INSERT INTO order_items (tenant_id, id_order, id_product, product_name_snapshot, unit_price_snapshot, quantity)
SELECT o3.tenant_id, o3.id_order, p.id_product, p.name, p.price, 1
FROM o3
JOIN products p ON p.tenant_id = 2 AND p.name = 'Tarta Negra Norte';

WITH o4 AS (
    INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, delivery_direction, paid, expires_at)
    SELECT
        2,
        u.id_user,
        'expired'::order_status,
        4.00,
        'mock_seed: t2 expired',
        '2026-03-15 09:00:00+00'::timestamptz,
        '2026-03-16'::date,
        'https://maps.app.goo.gl/mock-norte',
        FALSE,
        '2026-03-15 09:30:00+00'::timestamptz
    FROM users u
    WHERE u.tenant_id = 2 AND u.email = 'shared.client@demo.local'
      AND NOT EXISTS (SELECT 1 FROM orders WHERE tenant_id = 2 AND note = 'mock_seed: t2 expired')
    RETURNING id_order, tenant_id
)
INSERT INTO order_items (tenant_id, id_order, id_product, product_name_snapshot, unit_price_snapshot, quantity)
SELECT o4.tenant_id, o4.id_order, p.id_product, p.name, p.price, 1
FROM o4
JOIN products p ON p.tenant_id = 2 AND p.name = 'Pan Integral Norte';

WITH o5 AS (
    INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, delivery_direction, paid, expires_at)
    SELECT
        2,
        u.id_user,
        'preparing'::order_status,
        16.50,
        'mock_seed: t2 preparing',
        '2026-03-29 14:00:00+00'::timestamptz,
        '2026-03-31'::date,
        'https://maps.app.goo.gl/mock-norte',
        TRUE,
        NULL
    FROM users u
    WHERE u.tenant_id = 2 AND u.email = 'shared.client@demo.local'
      AND NOT EXISTS (SELECT 1 FROM orders WHERE tenant_id = 2 AND note = 'mock_seed: t2 preparing')
    RETURNING id_order, tenant_id
)
INSERT INTO order_items (tenant_id, id_order, id_product, product_name_snapshot, unit_price_snapshot, quantity)
SELECT o5.tenant_id, o5.id_order, p.id_product, p.name, p.price, 1
FROM o5
JOIN products p ON p.tenant_id = 2 AND p.name = 'Tarta Negra Norte';

WITH o6 AS (
    INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, delivery_direction, paid, expires_at)
    SELECT
        2,
        u.id_user,
        'cancelled'::order_status,
        8.00,
        'mock_seed: t2 cancelled',
        '2026-03-10 11:00:00+00'::timestamptz,
        '2026-03-12'::date,
        'https://maps.app.goo.gl/mock-norte',
        FALSE,
        NULL
    FROM users u
    WHERE u.tenant_id = 2 AND u.email = 'shared.client@demo.local'
      AND NOT EXISTS (SELECT 1 FROM orders WHERE tenant_id = 2 AND note = 'mock_seed: t2 cancelled')
    RETURNING id_order, tenant_id
)
INSERT INTO order_items (tenant_id, id_order, id_product, product_name_snapshot, unit_price_snapshot, quantity)
SELECT o6.tenant_id, o6.id_order, p.id_product, p.name, p.price, 2
FROM o6
JOIN products p ON p.tenant_id = 2 AND p.name = 'Pan Integral Norte';

-- Pedidos tenant 3
WITH c1 AS (
    INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, delivery_direction, paid, expires_at)
    SELECT
        3,
        u.id_user,
        'ready'::order_status,
        36.00,
        'mock_seed: t3 ready',
        '2026-03-28 16:00:00+00'::timestamptz,
        '2026-03-29'::date,
        'https://maps.app.goo.gl/mock-centro',
        TRUE,
        NULL
    FROM users u
    WHERE u.tenant_id = 3 AND u.email = 'shared.client@demo.local'
      AND NOT EXISTS (SELECT 1 FROM orders WHERE tenant_id = 3 AND note = 'mock_seed: t3 ready')
    RETURNING id_order, tenant_id
)
INSERT INTO order_items (tenant_id, id_order, id_product, product_name_snapshot, unit_price_snapshot, quantity)
SELECT c1.tenant_id, c1.id_order, p.id_product, p.name, p.price, 2
FROM c1
JOIN products p ON p.tenant_id = 3 AND p.name = 'Milhojas Centro';

WITH c2 AS (
    INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, delivery_direction, paid, expires_at)
    SELECT
        3,
        u.id_user,
        'delivered'::order_status,
        15.00,
        'mock_seed: t3 delivered',
        '2026-03-01 09:00:00+00'::timestamptz,
        '2026-03-02'::date,
        'https://maps.app.goo.gl/mock-centro',
        TRUE,
        NULL
    FROM users u
    WHERE u.tenant_id = 3 AND u.email = 'shared.client@demo.local'
      AND NOT EXISTS (SELECT 1 FROM orders WHERE tenant_id = 3 AND note = 'mock_seed: t3 delivered')
    RETURNING id_order, tenant_id
)
INSERT INTO order_items (tenant_id, id_order, id_product, product_name_snapshot, unit_price_snapshot, quantity)
SELECT c2.tenant_id, c2.id_order, p.id_product, p.name, p.price, 4
FROM c2
JOIN products p ON p.tenant_id = 3 AND p.name = 'Cupcake Centro';

WITH c3 AS (
    INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, delivery_direction, paid, expires_at)
    SELECT
        3,
        u.id_user,
        'pending'::order_status,
        18.00,
        'mock_seed: t3 pending cron',
        '2026-03-29 07:00:00+00'::timestamptz,
        '2026-03-30'::date,
        'https://maps.app.goo.gl/mock-centro',
        FALSE,
        '2026-03-29 07:25:00+00'::timestamptz
    FROM users u
    WHERE u.tenant_id = 3 AND u.email = 'shared.client@demo.local'
      AND NOT EXISTS (SELECT 1 FROM orders WHERE tenant_id = 3 AND note = 'mock_seed: t3 pending cron')
    RETURNING id_order, tenant_id
)
INSERT INTO order_items (tenant_id, id_order, id_product, product_name_snapshot, unit_price_snapshot, quantity)
SELECT c3.tenant_id, c3.id_order, p.id_product, p.name, p.price, 1
FROM c3
JOIN products p ON p.tenant_id = 3 AND p.name = 'Milhojas Centro';

WITH c4 AS (
    INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, delivery_direction, paid, expires_at)
    SELECT
        3,
        u.id_user,
        'preparing'::order_status,
        11.25,
        'mock_seed: t3 preparing',
        '2026-03-30 08:00:00+00'::timestamptz,
        '2026-03-31'::date,
        'https://maps.app.goo.gl/mock-centro',
        FALSE,
        '2026-03-30 08:40:00+00'::timestamptz
    FROM users u
    WHERE u.tenant_id = 3 AND u.email = 'shared.client@demo.local'
      AND NOT EXISTS (SELECT 1 FROM orders WHERE tenant_id = 3 AND note = 'mock_seed: t3 preparing')
    RETURNING id_order, tenant_id
)
INSERT INTO order_items (tenant_id, id_order, id_product, product_name_snapshot, unit_price_snapshot, quantity)
SELECT c4.tenant_id, c4.id_order, p.id_product, p.name, p.price, 3
FROM c4
JOIN products p ON p.tenant_id = 3 AND p.name = 'Cupcake Centro';

SELECT setval(pg_get_serial_sequence('tenants', 'id'), (SELECT COALESCE(MAX(id), 1) FROM tenants));
SELECT setval('users_id_user_seq', (SELECT COALESCE(MAX(id_user), 1) FROM users));
SELECT setval('products_id_product_seq', (SELECT COALESCE(MAX(id_product), 1) FROM products));
SELECT setval('products_history_id_products_history_seq', (SELECT COALESCE(MAX(id_products_history), 1) FROM products_history));
SELECT setval('orders_id_order_seq', (SELECT COALESCE(MAX(id_order), 1) FROM orders));
SELECT setval('order_items_id_order_item_seq', (SELECT COALESCE(MAX(id_order_item), 1) FROM order_items));
