-- Initialize SQL Schema (init_db.sql)

-- Users Table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY,
    language TEXT NOT NULL,
    help_type TEXT NOT NULL
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

-- Add indexes to queries table
CREATE INDEX IF NOT EXISTS idx_queries_language ON queries (language, help_type, word);

-- Cached Responses Table
CREATE TABLE IF NOT EXISTS cached_responses (
    query_id INTEGER NOT NULL,
    response TEXT NOT NULL,
    FOREIGN KEY (query_id) REFERENCES queries(id)
);
