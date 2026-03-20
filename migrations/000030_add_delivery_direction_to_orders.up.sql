-- Add delivery_direction to orders and orders_history

-- orders: add column, backfill existing rows, enforce NOT NULL
ALTER TABLE orders
    ADD COLUMN delivery_direction TEXT;

UPDATE orders
SET delivery_direction = 'https://maps.app.goo.gl/JewH99BXywGvtHQW6'
WHERE delivery_direction IS NULL;

ALTER TABLE orders
    ALTER COLUMN delivery_direction SET NOT NULL;

-- orders_history: add column, backfill existing rows, enforce NOT NULL
ALTER TABLE orders_history
    ADD COLUMN delivery_direction TEXT;

UPDATE orders_history
SET delivery_direction = 'https://maps.app.goo.gl/JewH99BXywGvtHQW6'
WHERE delivery_direction IS NULL;

ALTER TABLE orders_history
    ALTER COLUMN delivery_direction SET NOT NULL;

