ALTER TABLE products ADD COLUMN image_urls JSON AFTER status;
ALTER TABLE products_history ADD COLUMN image_urls JSON AFTER status;
