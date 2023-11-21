-- Initialize SQL Schema (init_db.sql)

-- Users Table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY,
    language TEXT NOT NULL
);

-- Queries Table
CREATE TABLE IF NOT EXISTS queries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    word TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Cached Responses Table
CREATE TABLE IF NOT EXISTS cached_responses (
    query_id INTEGER NOT NULL,
    response_type TEXT NOT NULL, -- e.g., 'usage', 'grammar'
    response TEXT NOT NULL,
    FOREIGN KEY (query_id) REFERENCES queries(id)
);
