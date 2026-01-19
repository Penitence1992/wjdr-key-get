-- Initial schema migration
-- Creates the base tables for users, gift code records, and tasks

-- Users table (fid_list)
CREATE TABLE IF NOT EXISTS fid_list (
    fid TEXT PRIMARY KEY,
    nickname TEXT NOT NULL,
    kid INTEGER NOT NULL,
    avatar_image TEXT NOT NULL
);

-- Gift code records table (gift_codes)
CREATE TABLE IF NOT EXISTS gift_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fid TEXT NOT NULL,
    code TEXT NOT NULL
);

-- Unique index for gift_codes
CREATE UNIQUE INDEX IF NOT EXISTS idx_fid_code ON gift_codes (fid, code);

-- Tasks table (gift_code_task)
CREATE TABLE IF NOT EXISTS gift_code_task (
    code TEXT PRIMARY KEY,
    all_done INTEGER NOT NULL
);
