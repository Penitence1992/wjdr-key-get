-- Migration: Create notifications table
-- Creates the notifications table for storing notification history with all required fields

-- Create notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    result TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('success', 'failed')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for querying notifications by creation time (newest first)
CREATE INDEX IF NOT EXISTS idx_notification_created ON notifications(created_at DESC);
