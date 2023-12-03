-- Initialize SQL Schema (init_db.sql)

-- Users Table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY,
    language TEXT NOT NULL,
    help_type TEXT NOT NULL,
    speech_speed REAL NOT NULL DEFAULT 0.0 -- Set a default value for the speech_speed column
);

-- Queries Table
CREATE TABLE IF NOT EXISTS queries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    word TEXT NOT NULL,
    language TEXT NOT NULL,
    help_type TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Add column speech_speed to users table if it doesn't exist
CREATE TABLE IF NOT EXISTS temp_users AS SELECT * FROM users; -- Create a temporary table
DROP TABLE IF EXISTS users; -- Drop the original users table
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    language TEXT NOT NULL,
    help_type TEXT NOT NULL,
    speech_speed REAL NOT NULL DEFAULT 0.0 -- Recreate the users table with the new column
);
INSERT INTO users (id, language, help_type) SELECT id, language, help_type FROM temp_users;

DROP TABLE IF EXISTS temp_users; -- Drop the temporary table

-- Add indexes to queries table
CREATE INDEX IF NOT EXISTS idx_queries_language ON queries (language, help_type, word);

-- Cached Responses Table
CREATE TABLE IF NOT EXISTS cached_responses (
    query_id INTEGER NOT NULL,
    response TEXT NOT NULL,
    FOREIGN KEY (query_id) REFERENCES queries(id)
);

