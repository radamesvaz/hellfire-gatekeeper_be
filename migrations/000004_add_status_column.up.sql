-- Create status enum type for PostgreSQL (if not exists)
DO $$ BEGIN
    CREATE TYPE product_status AS ENUM ('active', 'inactive', 'deleted');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Modify the products table to add a status column (if not exists)
DO $$ BEGIN
    ALTER TABLE products ADD COLUMN status product_status NOT NULL DEFAULT 'active';
EXCEPTION
    WHEN duplicate_column THEN null;
END $$;

-- Modify the products_history table to add a status column (if not exists)
DO $$ BEGIN
    ALTER TABLE products_history ADD COLUMN status product_status NOT NULL DEFAULT 'active';
EXCEPTION
    WHEN duplicate_column THEN null;
END $$;