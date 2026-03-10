-- Index on expires_at for ghost-order cron.
-- Only pending and unpaid orders are considered, matching the cron filter.

CREATE INDEX idx_orders_expires_at_pending
ON orders (expires_at)
WHERE status = 'pending' AND paid = false;

