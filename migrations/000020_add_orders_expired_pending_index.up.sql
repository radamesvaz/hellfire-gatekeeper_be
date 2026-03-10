-- Composite index for GetExpiredPendingOrders (cron: ghost orders).
-- Partial index: only rows that can match the query (pending, unpaid), then range on created_on.
CREATE INDEX idx_orders_expired_pending
ON orders (status, paid, created_on)
WHERE status = 'pending' AND paid = false;
