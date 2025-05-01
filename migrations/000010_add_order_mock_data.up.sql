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


ALTER TABLE order_items DROP FOREIGN KEY order_items_ibfk_1;

ALTER TABLE order_items DROP PRIMARY KEY;

ALTER TABLE order_items CHANGE id_order_items id_order_item INT NOT NULL;

ALTER TABLE order_items ADD COLUMN id_order INT NOT NULL AFTER id_order_item;

ALTER TABLE order_items MODIFY id_order_item INT NOT NULL AUTO_INCREMENT, ADD PRIMARY KEY (id_order_item);

ALTER TABLE order_items
  ADD CONSTRAINT fk_order FOREIGN KEY (id_order) REFERENCES orders(id_order) ON DELETE CASCADE,
  ADD CONSTRAINT fk_product FOREIGN KEY (id_product) REFERENCES products(id_product) ON DELETE CASCADE;


DELETE FROM order_items where id_order_item IN (1,2,3,4,5);
INSERT INTO order_items (id_order_item, id_product, id_order, quantity) 
    VALUES
    (1, 1, 1, 2),
    (2, 2, 1, 10),
    (3, 2, 2, 2),
    (4, 1, 3, 2),
    (5, 2, 3, 1);