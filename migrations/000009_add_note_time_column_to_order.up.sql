ALTER TABLE orders ADD COLUMN note TEXT AFTER total_price;
ALTER TABLE orders ADD COLUMN delivery_date DATE;

ALTER TABLE orders_history ADD COLUMN note TEXT AFTER total_price;
ALTER TABLE orders_history ADD COLUMN delivery_date DATE;