-- Rollback initial schema migration
-- Drops all tables created in the up migration

DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS gift_code_records;
DROP TABLE IF EXISTS users;
