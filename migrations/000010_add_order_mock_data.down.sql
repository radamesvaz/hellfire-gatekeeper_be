ALTER TABLE order_items DROP FOREIGN KEY fk_order;
ALTER TABLE order_items DROP FOREIGN KEY fk_product;

ALTER TABLE order_items DROP COLUMN id_order;

ALTER TABLE order_items MODIFY id_order_item INT NOT NULL;

ALTER TABLE order_items DROP PRIMARY KEY;

ALTER TABLE order_items ADD PRIMARY KEY (id_order_item, id_product);

ALTER TABLE order_items
ADD CONSTRAINT order_items_ibfk_1 FOREIGN KEY (id_order_item)
REFERENCES orders(id_order) ON DELETE CASCADE;
