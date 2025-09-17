-- Drop the foreign key constraint
ALTER TABLE products_history
DROP CONSTRAINT fk_products_history_modified_by;

-- Note: modified_by and action columns remain as they were