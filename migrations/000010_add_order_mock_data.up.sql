-- DELETE FROM orders_history WHERE id_order IN (1, 2, 3, 4);
DELETE FROM orders WHERE id_order IN (1, 2, 3, 4);

INSERT INTO orders (id_order, id_user, status, total_price, note, created_on, delivery_date)
VALUES
  (1, 2, 'delivered', 40, 'make it bright', '2025-04-1 10:00:00', '2025-04-5 10:00:00'),
  (2, 2, 'pending', 20, 'deliver at the door', '2025-04-14 10:00:00', '2025-04-20 10:00:00'),
  (3, 2, 'preparing', 40, 'not so sweet', '2025-04-20 10:00:00', '2025-04-25 10:00:00'),
  (4, 2, 'cancelled', 40, 'this one is canceled', '2025-04-20 10:00:00', '2025-04-25 10:00:00');


-- INSERT INTO orders_history (id_orders_history, id_order, id_user, status, total_price, note, modified_on, modified_by, action, delivery_date)
-- VALUES
--   (1, 1, 2, 'delivered', 40, 'make it bright', '2025-04-1 10:00:00', '2025-04-5 10:00:00'),
--   (2, 2, 'pending', 20, 'deliver at the door', '2025-04-14 10:00:00', '2025-04-20 10:00:00'),
--   (3, 3, 'preparing', 40, 'not so sweet', '2025-04-20 10:00:00', '2025-04-25 10:00:00'),
--   (4, 4, 'cancelled', 40, 'this one is canceled', '2025-04-20 10:00:00', '2025-04-25 10:00:00');


-- Drop existing constraints and recreate for PostgreSQL
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_ibfk_1;
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS fk_order;
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS fk_product;

-- Drop and recreate the table with proper structure
DROP TABLE IF EXISTS order_items;

CREATE TABLE order_items (
    id_order_item SERIAL PRIMARY KEY,
    id_order INT NOT NULL,
    id_product INT NOT NULL,
    quantity INT NOT NULL,
    CONSTRAINT fk_order FOREIGN KEY (id_order) REFERENCES orders(id_order) ON DELETE CASCADE,
    CONSTRAINT fk_product FOREIGN KEY (id_product) REFERENCES products(id_product) ON DELETE CASCADE
);

INSERT INTO order_items (id_order_item, id_product, id_order, quantity) 
    VALUES
    (1, 1, 1, 2),
    (2, 2, 1, 10),
    (3, 2, 2, 2),
    (4, 1, 3, 2),
    (5, 2, 3, 1);