CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_name TEXT NOT NULL,
    genre TEXT NOT NULL,
    age INTEGER,
    gender TEXT,
    archetype TEXT,
    seed TEXT,
    state TEXT NOT NULL,
    current_choices TEXT,
    status TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS story_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    turn_number INTEGER NOT NULL,
    content TEXT NOT NULL,
    chapter_title TEXT,
    choice_made TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(session_id) REFERENCES sessions(id)
);

CREATE INDEX IF NOT EXISTS idx_story_logs_session_id ON story_logs(session_id, turn_number);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at);
