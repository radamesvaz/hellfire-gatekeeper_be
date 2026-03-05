-- Hacer historiales inmutables y conservar órdenes al borrar usuario.
-- orders: id_user nullable, ON DELETE SET NULL (no borrar órdenes al borrar usuario)
-- products_history: id_product nullable, ON DELETE SET NULL (conservar historial si se borra producto)
-- orders_history: id_order e id_user nullable, ON DELETE SET NULL (conservar historial)

-- 1. orders: permitir NULL en id_user y cambiar FK a SET NULL
ALTER TABLE orders ALTER COLUMN id_user DROP NOT NULL;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_id_user_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_id_user_fkey
    FOREIGN KEY (id_user) REFERENCES users(id_user) ON DELETE SET NULL;

-- 2. products_history: permitir NULL en id_product y cambiar FK a SET NULL
ALTER TABLE products_history ALTER COLUMN id_product DROP NOT NULL;
ALTER TABLE products_history DROP CONSTRAINT IF EXISTS products_history_id_product_fkey;
ALTER TABLE products_history ADD CONSTRAINT products_history_id_product_fkey
    FOREIGN KEY (id_product) REFERENCES products(id_product) ON DELETE SET NULL;

-- 3. orders_history: permitir NULL en id_order e id_user y cambiar ambas FKs a SET NULL
ALTER TABLE orders_history ALTER COLUMN id_order DROP NOT NULL;
ALTER TABLE orders_history ALTER COLUMN id_user DROP NOT NULL;
ALTER TABLE orders_history DROP CONSTRAINT IF EXISTS orders_history_id_order_fkey;
ALTER TABLE orders_history DROP CONSTRAINT IF EXISTS orders_history_id_user_fkey;
ALTER TABLE orders_history ADD CONSTRAINT orders_history_id_order_fkey
    FOREIGN KEY (id_order) REFERENCES orders(id_order) ON DELETE SET NULL;
ALTER TABLE orders_history ADD CONSTRAINT orders_history_id_user_fkey
    FOREIGN KEY (id_user) REFERENCES users(id_user) ON DELETE SET NULL;
