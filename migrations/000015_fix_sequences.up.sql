-- Fix sequences to match existing data
SELECT setval('products_id_product_seq', (SELECT MAX(id_product) FROM products));
SELECT setval('users_id_user_seq', (SELECT MAX(id_user) FROM users));
SELECT setval('orders_id_order_seq', (SELECT MAX(id_order) FROM orders));
SELECT setval('order_items_id_order_item_seq', (SELECT MAX(id_order_item) FROM order_items));
SELECT setval('products_history_id_products_history_seq', (SELECT MAX(id_products_history) FROM products_history));
SELECT setval('orders_history_id_order_history_seq', (SELECT MAX(id_order_history) FROM orders_history));


