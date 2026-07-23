ALTER TABLE products
    ADD COLUMN track_inventory BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE products_history
    ADD COLUMN track_inventory BOOLEAN NOT NULL DEFAULT true;
