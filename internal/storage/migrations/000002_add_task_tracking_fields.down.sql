-- Migration rollback: Remove task tracking fields from gift_code_task table
-- Removes the fields added in 000002_add_task_tracking_fields.up.sql

-- Drop the index first
DROP INDEX IF EXISTS idx_task_completed;

-- Drop the added columns in reverse order
ALTER TABLE gift_code_task DROP COLUMN last_error;
ALTER TABLE gift_code_task DROP COLUMN retry_count;
ALTER TABLE gift_code_task DROP COLUMN completed_at;
ALTER TABLE gift_code_task DROP COLUMN created_at;
