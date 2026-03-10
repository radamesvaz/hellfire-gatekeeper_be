-- Add expires_at to orders to store per-order expiration time for ghost orders.
-- NOTE: today the timeout is global (GHOST_ORDER_TIMEOUT_MINUTES env). In a future
-- multi-tenant setup, expires_at should be computed using the per-tenant config.

ALTER TABLE orders
    ADD COLUMN expires_at TIMESTAMPTZ;

-- Backfill for existing pending/unpaid orders so the cron can start using expires_at.
-- We use a fixed 30 minutes window, aligned with the current default timeout. If the
-- env value changes in the future, only new orders will use the new timeout.
UPDATE orders
SET expires_at = created_on + INTERVAL '30 minutes'
WHERE status = 'pending'
  AND paid = false
  AND expires_at IS NULL;

