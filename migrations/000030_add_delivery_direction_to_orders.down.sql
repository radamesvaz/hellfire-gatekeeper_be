-- Revert delivery_direction changes on orders_history
ALTER TABLE orders_history
    DROP COLUMN IF EXISTS delivery_direction;

-- Revert delivery_direction changes on orders
ALTER TABLE orders
    DROP COLUMN IF EXISTS delivery_direction;

