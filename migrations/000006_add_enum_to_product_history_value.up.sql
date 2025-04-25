ALTER TABLE products_history
MODIFY action ENUM('create', 'update', 'delete');