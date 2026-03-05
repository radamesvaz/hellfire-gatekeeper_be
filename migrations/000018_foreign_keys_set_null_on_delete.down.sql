-- Revertir a ON DELETE CASCADE. Las columnas se dejan nullable porque
-- tras ejecutar la migración up pueden existir NULLs (usuarios/órdenes/productos borrados).

-- 1. orders
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_id_user_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_id_user_fkey
    FOREIGN KEY (id_user) REFERENCES users(id_user) ON DELETE CASCADE;

-- 2. products_history
ALTER TABLE products_history DROP CONSTRAINT IF EXISTS products_history_id_product_fkey;
ALTER TABLE products_history ADD CONSTRAINT products_history_id_product_fkey
    FOREIGN KEY (id_product) REFERENCES products(id_product) ON DELETE CASCADE;

-- 3. orders_history
ALTER TABLE orders_history DROP CONSTRAINT IF EXISTS orders_history_id_order_fkey;
ALTER TABLE orders_history DROP CONSTRAINT IF EXISTS orders_history_id_user_fkey;
ALTER TABLE orders_history ADD CONSTRAINT orders_history_id_order_fkey
    FOREIGN KEY (id_order) REFERENCES orders(id_order) ON DELETE CASCADE;
ALTER TABLE orders_history ADD CONSTRAINT orders_history_id_user_fkey
    FOREIGN KEY (id_user) REFERENCES users(id_user) ON DELETE CASCADE;
