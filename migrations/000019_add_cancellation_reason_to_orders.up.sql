ALTER TABLE orders ADD COLUMN cancellation_reason VARCHAR(255) NULL;

ALTER TABLE orders_history ADD COLUMN cancellation_reason VARCHAR(255) NULL;
