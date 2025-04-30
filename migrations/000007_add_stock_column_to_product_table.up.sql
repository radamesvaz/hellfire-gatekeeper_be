ALTER TABLE products ADD COLUMN stock INT NOT NULL DEFAULT 0 AFTER available;
ALTER TABLE products_history ADD COLUMN stock INT NOT NULL DEFAULT 0 AFTER available;

UPDATE products SET stock=1 WHERE id_product=1;
UPDATE products_history SET stock=1 WHERE id_product=1;

