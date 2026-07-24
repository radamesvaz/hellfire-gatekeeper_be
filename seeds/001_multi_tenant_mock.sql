-- Local dev only: datos multi-tenant de prueba (idempotente).
-- Ejecutar tras `migrate up`. Ver seeds/README.md

-- bcrypt cost 10: adminpass
-- bcrypt cost 10: clientpass

UPDATE users
SET password_hash = '$2a$10$Ued/UmuZslCPE.c.iRdFEON7jicWeAlBBM3vjLrcktSg778XHtvQW'
WHERE id_role = (SELECT id_role FROM roles WHERE name = 'admin');

UPDATE users
SET password_hash = '$2a$10$Ued/UmuZslCPE.c.iRdFEON7jicWeAlBBM3vjLrcktSg778XHtvQW'
WHERE id_role = (SELECT id_role FROM roles WHERE name = 'superadmin');

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

INSERT INTO products (name, description, price, track_inventory, created_on, status, tenant_id, stock, image_urls, thumbnail_url)
SELECT 'Tarta Negra Norte', 'Mock tarta chocolate', 12.50, TRUE, NOW(), 'active', 2, 20, NULL, NULL
WHERE NOT EXISTS (SELECT 1 FROM products WHERE tenant_id = 2 AND name = 'Tarta Negra Norte');

INSERT INTO products (name, description, price, track_inventory, created_on, status, tenant_id, stock, image_urls, thumbnail_url)
SELECT 'Pan Integral Norte', 'Mock pan integral', 4.00, TRUE, NOW(), 'active', 2, 50, NULL, NULL
WHERE NOT EXISTS (SELECT 1 FROM products WHERE tenant_id = 2 AND name = 'Pan Integral Norte');

INSERT INTO products (name, description, price, track_inventory, created_on, status, tenant_id, stock, image_urls, thumbnail_url)
SELECT 'Milhojas Centro', 'Mock milhojas', 18.00, TRUE, NOW(), 'active', 3, 15, NULL, NULL
WHERE NOT EXISTS (SELECT 1 FROM products WHERE tenant_id = 3 AND name = 'Milhojas Centro');

INSERT INTO products (name, description, price, track_inventory, created_on, status, tenant_id, stock, image_urls, thumbnail_url)
SELECT 'Cupcake Centro', 'Mock cupcake', 3.75, TRUE, NOW(), 'active', 3, 40, NULL, NULL
WHERE NOT EXISTS (SELECT 1 FROM products WHERE tenant_id = 3 AND name = 'Cupcake Centro');

INSERT INTO products_history (
    tenant_id, id_product, name, description, price, track_inventory, stock, status, image_urls, thumbnail_url, modified_by, action
)
SELECT
    p.tenant_id,
    p.id_product,
    p.name,
    p.description,
    p.price,
    p.track_inventory,
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

-- Volumen extra para probar paginación en frontend (limit default 20).
-- Tenant 1 (default): catálogo y pedidos GET /products y GET /auth/orders.
-- Tenants 2–3: catálogo GET /t/{slug}/products.

INSERT INTO products (name, description, price, track_inventory, created_on, status, tenant_id, stock, image_urls, thumbnail_url)
SELECT
    'seed_pagination_t1_' || lpad(n::text, 3, '0'),
    'Producto de seed para probar paginación (tenant default)',
    (1.00 + (n % 18) * 0.25)::numeric,
    TRUE,
    NOW(),
    'active',
    1,
    25,
    NULL,
    NULL
FROM generate_series(1, 45) AS n
WHERE NOT EXISTS (
    SELECT 1 FROM products p
    WHERE p.tenant_id = 1 AND p.name = 'seed_pagination_t1_' || lpad(n::text, 3, '0')
);

INSERT INTO products (name, description, price, track_inventory, created_on, status, tenant_id, stock, image_urls, thumbnail_url)
SELECT
    'seed_pagination_t2_' || lpad(n::text, 3, '0'),
    'Producto de seed para probar paginación (Panadería Norte)',
    (2.00 + (n % 15) * 0.5)::numeric,
    TRUE,
    NOW(),
    'active',
    2,
    30,
    NULL,
    NULL
FROM generate_series(1, 28) AS n
WHERE NOT EXISTS (
    SELECT 1 FROM products p
    WHERE p.tenant_id = 2 AND p.name = 'seed_pagination_t2_' || lpad(n::text, 3, '0')
);

INSERT INTO products (name, description, price, track_inventory, created_on, status, tenant_id, stock, image_urls, thumbnail_url)
SELECT
    'seed_pagination_t3_' || lpad(n::text, 3, '0'),
    'Producto de seed para probar paginación (Pastelería Centro)',
    (3.00 + (n % 12) * 0.5)::numeric,
    TRUE,
    NOW(),
    'active',
    3,
    20,
    NULL,
    NULL
FROM generate_series(1, 28) AS n
WHERE NOT EXISTS (
    SELECT 1 FROM products p
    WHERE p.tenant_id = 3 AND p.name = 'seed_pagination_t3_' || lpad(n::text, 3, '0')
);

INSERT INTO products_history (
    tenant_id, id_product, name, description, price, track_inventory, stock, status, image_urls, thumbnail_url, modified_by, action
)
SELECT
    p.tenant_id,
    p.id_product,
    p.name,
    p.description,
    p.price,
    p.track_inventory,
    p.stock,
    p.status,
    p.image_urls,
    p.thumbnail_url,
    u.id_user,
    'create'::history_action
FROM products p
JOIN users u ON u.tenant_id = p.tenant_id AND u.id_role = (SELECT id_role FROM roles WHERE name = 'admin')
WHERE p.tenant_id IN (1, 2, 3)
  AND p.name LIKE 'seed_pagination_t%'
  AND NOT EXISTS (
      SELECT 1 FROM products_history ph
      WHERE ph.id_product = p.id_product AND ph.tenant_id = p.tenant_id AND ph.action = 'create'::history_action
  );

-- Pedidos tenant 1 (cliente client@example.com): notas únicas para idempotencia.
WITH to_insert AS (
    SELECT
        n,
        'mock_seed_pagination_order_t1_' || lpad(n::text, 3, '0') AS note_key
    FROM generate_series(1, 35) AS n
    WHERE NOT EXISTS (
        SELECT 1 FROM orders o
        WHERE o.tenant_id = 1 AND o.note = 'mock_seed_pagination_order_t1_' || lpad(n::text, 3, '0')
    )
)
INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, delivery_direction, paid, expires_at)
SELECT
    1,
    (SELECT id_user FROM users WHERE tenant_id = 1 AND email = 'client@example.com' LIMIT 1),
    (ARRAY[
        'pending'::order_status,
        'preparing'::order_status,
        'ready'::order_status,
        'delivered'::order_status,
        'cancelled'::order_status
    ])[1 + (t.n % 5)],
    (12.00 + (t.n % 20))::numeric,
    t.note_key,
    ('2026-06-01 08:00:00+00'::timestamptz + (t.n || ' minutes')::interval),
    ('2026-06-02'::date + t.n),
    'https://maps.app.goo.gl/mock-default',
    (t.n % 3 <> 0),
    NULL
FROM to_insert t;

INSERT INTO order_items (tenant_id, id_order, id_product, product_name_snapshot, unit_price_snapshot, quantity)
SELECT
    o.tenant_id,
    o.id_order,
    p.id_product,
    p.name,
    p.price,
    1 + (o.id_order % 3)
FROM orders o
JOIN LATERAL (
    SELECT id_product, name, price
    FROM products
    WHERE tenant_id = 1 AND status = 'active'
    ORDER BY id_product
    OFFSET 0
    LIMIT 1
) p ON TRUE
WHERE o.tenant_id = 1
  AND o.note LIKE 'mock_seed_pagination_order_t1_%'
  AND NOT EXISTS (SELECT 1 FROM order_items oi WHERE oi.id_order = o.id_order);

WITH to_insert2 AS (
    SELECT
        n,
        'mock_seed_pagination_order_t2_' || lpad(n::text, 3, '0') AS note_key
    FROM generate_series(1, 28) AS n
    WHERE NOT EXISTS (
        SELECT 1 FROM orders o
        WHERE o.tenant_id = 2 AND o.note = 'mock_seed_pagination_order_t2_' || lpad(n::text, 3, '0')
    )
)
INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, delivery_direction, paid, expires_at)
SELECT
    2,
    (SELECT id_user FROM users WHERE tenant_id = 2 AND email = 'shared.client@demo.local' LIMIT 1),
    (ARRAY['pending'::order_status, 'delivered'::order_status, 'preparing'::order_status])[1 + (t.n % 3)],
    (8.00 + (t.n % 15))::numeric,
    t.note_key,
    ('2026-06-10 09:00:00+00'::timestamptz + (t.n || ' minutes')::interval),
    ('2026-06-11'::date + t.n),
    'https://maps.app.goo.gl/mock-norte',
    TRUE,
    NULL
FROM to_insert2 t;

INSERT INTO order_items (tenant_id, id_order, id_product, product_name_snapshot, unit_price_snapshot, quantity)
SELECT
    o.tenant_id,
    o.id_order,
    p.id_product,
    p.name,
    p.price,
    1
FROM orders o
JOIN LATERAL (
    SELECT id_product, name, price
    FROM products
    WHERE tenant_id = 2 AND status = 'active'
    ORDER BY id_product DESC
    LIMIT 1
) p ON TRUE
WHERE o.tenant_id = 2
  AND o.note LIKE 'mock_seed_pagination_order_t2_%'
  AND NOT EXISTS (SELECT 1 FROM order_items oi WHERE oi.id_order = o.id_order);

WITH to_insert3 AS (
    SELECT
        n,
        'mock_seed_pagination_order_t3_' || lpad(n::text, 3, '0') AS note_key
    FROM generate_series(1, 28) AS n
    WHERE NOT EXISTS (
        SELECT 1 FROM orders o
        WHERE o.tenant_id = 3 AND o.note = 'mock_seed_pagination_order_t3_' || lpad(n::text, 3, '0')
    )
)
INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, delivery_direction, paid, expires_at)
SELECT
    3,
    (SELECT id_user FROM users WHERE tenant_id = 3 AND email = 'shared.client@demo.local' LIMIT 1),
    (ARRAY['pending'::order_status, 'ready'::order_status, 'delivered'::order_status])[1 + (t.n % 3)],
    (10.00 + (t.n % 18))::numeric,
    t.note_key,
    ('2026-06-20 10:00:00+00'::timestamptz + (t.n || ' minutes')::interval),
    ('2026-06-21'::date + t.n),
    'https://maps.app.goo.gl/mock-centro',
    (t.n % 2 = 0),
    NULL
FROM to_insert3 t;

INSERT INTO order_items (tenant_id, id_order, id_product, product_name_snapshot, unit_price_snapshot, quantity)
SELECT
    o.tenant_id,
    o.id_order,
    p.id_product,
    p.name,
    p.price,
    2
FROM orders o
JOIN LATERAL (
    SELECT id_product, name, price
    FROM products
    WHERE tenant_id = 3 AND status = 'active'
    ORDER BY id_product ASC
    LIMIT 1
) p ON TRUE
WHERE o.tenant_id = 3
  AND o.note LIKE 'mock_seed_pagination_order_t3_%'
  AND NOT EXISTS (SELECT 1 FROM order_items oi WHERE oi.id_order = o.id_order);

SELECT setval(pg_get_serial_sequence('tenants', 'id'), (SELECT COALESCE(MAX(id), 1) FROM tenants));
SELECT setval('users_id_user_seq', (SELECT COALESCE(MAX(id_user), 1) FROM users));
SELECT setval('products_id_product_seq', (SELECT COALESCE(MAX(id_product), 1) FROM products));
SELECT setval('products_history_id_products_history_seq', (SELECT COALESCE(MAX(id_products_history), 1) FROM products_history));
SELECT setval('orders_id_order_seq', (SELECT COALESCE(MAX(id_order), 1) FROM orders));
SELECT setval('order_items_id_order_item_seq', (SELECT COALESCE(MAX(id_order_item), 1) FROM order_items));
