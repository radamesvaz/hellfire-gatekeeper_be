-- Create status enum type for PostgreSQL
CREATE TYPE product_status AS ENUM ('active', 'inactive', 'deleted');

-- Modify the products table to add a status column
ALTER TABLE products ADD COLUMN status product_status NOT NULL DEFAULT 'active';

-- Modify the products_history table to add a status column
ALTER TABLE products_history ADD COLUMN status product_status NOT NULL DEFAULT 'active';