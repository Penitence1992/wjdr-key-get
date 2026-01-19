-- Migration rollback: Drop notifications table
-- Removes the notifications table and its index

-- Drop the index first
DROP INDEX IF EXISTS idx_notification_created;

-- Drop the notifications table
DROP TABLE IF EXISTS notifications;
