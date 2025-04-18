-- Modify the products table to add a status column
ALTER TABLE products ADD COLUMN status ENUM('active', 'inactive', 'deleted') NOT NULL DEFAULT 'active' AFTER available;

-- Modify the products_history table to add a status column
ALTER TABLE products_history ADD COLUMN status ENUM('active', 'inactive', 'deleted') NOT NULL DEFAULT 'active' AFTER available;