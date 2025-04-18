ALTER TABLE products_history
DROP FOREIGN KEY fk_products_history_modified_by;

ALTER TABLE products_history
MODIFY modified_by INT NULL;

ALTER TABLE products_history
MODIFY action ENUM('update', 'delete') NOT NULL;