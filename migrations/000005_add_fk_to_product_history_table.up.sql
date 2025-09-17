-- First, add the foreign key constraint
ALTER TABLE products_history
ADD CONSTRAINT fk_products_history_modified_by
FOREIGN KEY (modified_by) REFERENCES users(id_user);

-- Note: action column already exists as history_action type from initial schema