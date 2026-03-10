-- Add 'expired' status to order_status enum for ghost orders.
-- NOTE: ALTER TYPE ... ADD VALUE is not easily reversible and cannot run inside a transaction
-- in older PostgreSQL versions. This migration is intentionally one-way.

ALTER TYPE order_status ADD VALUE IF NOT EXISTS 'expired';

