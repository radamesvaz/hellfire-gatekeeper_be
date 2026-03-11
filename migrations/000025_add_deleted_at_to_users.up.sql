-- Soft delete support for users: add deleted_at column.
-- Users will no longer be removed physically; instead, deleted_at will be set and
-- queries that care about active users must filter on deleted_at IS NULL.

ALTER TABLE users
    ADD COLUMN deleted_at TIMESTAMPTZ;

