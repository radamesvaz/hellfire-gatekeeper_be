ALTER TABLE orders ALTER COLUMN id_user DROP NOT NULL;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_id_user_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_id_user_fkey
    FOREIGN KEY (id_user) REFERENCES users(id_user) ON DELETE SET NULL;

ALTER TABLE products_history ALTER COLUMN id_product DROP NOT NULL;
ALTER TABLE products_history DROP CONSTRAINT IF EXISTS products_history_id_product_fkey;
ALTER TABLE products_history ADD CONSTRAINT products_history_id_product_fkey
    FOREIGN KEY (id_product) REFERENCES products(id_product) ON DELETE SET NULL;

ALTER TABLE orders_history ALTER COLUMN id_order DROP NOT NULL;
ALTER TABLE orders_history ALTER COLUMN id_user DROP NOT NULL;
ALTER TABLE orders_history DROP CONSTRAINT IF EXISTS orders_history_id_order_fkey;
ALTER TABLE orders_history DROP CONSTRAINT IF EXISTS orders_history_id_user_fkey;
ALTER TABLE orders_history ADD CONSTRAINT orders_history_id_order_fkey
    FOREIGN KEY (id_order) REFERENCES orders(id_order) ON DELETE SET NULL;
ALTER TABLE orders_history ADD CONSTRAINT orders_history_id_user_fkey
    FOREIGN KEY (id_user) REFERENCES users(id_user) ON DELETE SET NULL;
