ALTER TABLE products_history
MODIFY modified_by INT NOT NULL,
ADD CONSTRAINT fk_products_history_modified_by
FOREIGN KEY (modified_by) REFERENCES users(id_user);

ALTER TABLE products_history
MODIFY action ENUM('create', 'update', 'delete') NOT NULL;