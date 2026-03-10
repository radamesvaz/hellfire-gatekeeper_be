ALTER TABLE order_items
    DROP COLUMN IF EXISTS product_name_snapshot,
    DROP COLUMN IF EXISTS unit_price_snapshot;

