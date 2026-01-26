ALTER TABLE products ADD COLUMN thumbnail_url TEXT;
ALTER TABLE products_history ADD COLUMN thumbnail_url TEXT;

-- Backfill thumbnail_url using the first image in image_urls, when present
UPDATE products
SET thumbnail_url = image_urls ->> 0
WHERE thumbnail_url IS NULL
  AND image_urls IS NOT NULL
  AND json_array_length(image_urls) > 0;

UPDATE products_history
SET thumbnail_url = image_urls ->> 0
WHERE thumbnail_url IS NULL
  AND image_urls IS NOT NULL
  AND json_array_length(image_urls) > 0;
