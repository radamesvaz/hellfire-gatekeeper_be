ALTER TABLE orders_history
MODIFY action ENUM('create', 'update', 'delete');
