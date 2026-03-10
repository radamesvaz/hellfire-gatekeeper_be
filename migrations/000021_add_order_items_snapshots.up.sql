-- Add snapshot columns to order_items to capture product name and unit price at order time.

ALTER TABLE order_items
    ADD COLUMN product_name_snapshot VARCHAR(255),
    ADD COLUMN unit_price_snapshot  DECIMAL(10,2);

-- Backfill existing order_items with current product data so historical orders remain consistent.
UPDATE order_items oi
SET
    product_name_snapshot = p.name,
    unit_price_snapshot   = p.price
FROM products p
WHERE oi.id_product = p.id_product
  AND (oi.product_name_snapshot IS NULL OR oi.unit_price_snapshot IS NULL);

