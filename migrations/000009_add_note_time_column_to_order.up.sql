ALTER TABLE orders ADD COLUMN note TEXT;
ALTER TABLE orders ADD COLUMN delivery_date DATE;

ALTER TABLE orders_history ADD COLUMN note TEXT;
ALTER TABLE orders_history ADD COLUMN delivery_date DATE;