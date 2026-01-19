-- Migration: Add task tracking fields to gift_code_task table
-- Adds created_at, completed_at, retry_count, and last_error fields for enhanced task monitoring

-- Add created_at field (nullable first, will set default values after)
ALTER TABLE gift_code_task ADD COLUMN created_at TIMESTAMP;

-- Add completed_at field (nullable, set when task completes)
ALTER TABLE gift_code_task ADD COLUMN completed_at TIMESTAMP;

-- Add retry_count field with default 0
ALTER TABLE gift_code_task ADD COLUMN retry_count INTEGER DEFAULT 0;

-- Add last_error field with default empty string
ALTER TABLE gift_code_task ADD COLUMN last_error TEXT DEFAULT '';

-- Set default values for existing records
UPDATE gift_code_task SET created_at = CURRENT_TIMESTAMP WHERE created_at IS NULL;

-- Create index for querying completed tasks efficiently
CREATE INDEX IF NOT EXISTS idx_task_completed ON gift_code_task(all_done, completed_at DESC);
