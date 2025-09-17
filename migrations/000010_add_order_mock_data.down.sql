-- Drop foreign key constraints
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS fk_order;
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS fk_product;

-- Drop the id_order column
ALTER TABLE order_items DROP COLUMN IF EXISTS id_order;

-- Note: id_order_item remains as SERIAL (auto-increment) from the up migration

-- Drop primary key
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_pkey;

-- Add composite primary key
ALTER TABLE order_items ADD PRIMARY KEY (id_order_item, id_product);

-- Add foreign key constraint
ALTER TABLE order_items
ADD CONSTRAINT order_items_ibfk_1 FOREIGN KEY (id_order_item)
REFERENCES orders(id_order) ON DELETE CASCADE;
