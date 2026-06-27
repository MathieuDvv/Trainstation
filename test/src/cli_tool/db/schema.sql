PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS language (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    code        TEXT    NOT NULL UNIQUE,
    name        TEXT    NOT NULL,
    flag_emoji  TEXT
);

CREATE TABLE IF NOT EXISTS vocab_list (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT    NOT NULL,
    description     TEXT,
    source_lang_id  INTEGER NOT NULL REFERENCES language(id),
    target_lang_id  INTEGER NOT NULL REFERENCES language(id),
    topic           TEXT,
    cefr_level      TEXT,
    is_builtin      INTEGER DEFAULT 0,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS vocab_term (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    list_id         INTEGER NOT NULL REFERENCES vocab_list(id) ON DELETE CASCADE,
    source_word     TEXT    NOT NULL,
    target_word     TEXT    NOT NULL,
    part_of_speech  TEXT,
    gender          TEXT,
    example_sentence TEXT,
    pronunciation   TEXT,
    easiness_factor REAL    NOT NULL DEFAULT 2.5,
    interval_days   INTEGER NOT NULL DEFAULT 0,
    repetitions     INTEGER NOT NULL DEFAULT 0,
    next_review     TEXT    NOT NULL DEFAULT (date('now')),
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    last_reviewed   TEXT,
    UNIQUE(list_id, source_word)
);

CREATE INDEX IF NOT EXISTS idx_vocab_term_list     ON vocab_term(list_id);
CREATE INDEX IF NOT EXISTS idx_vocab_term_due      ON vocab_term(next_review, list_id);
CREATE INDEX IF NOT EXISTS idx_vocab_term_mastery  ON vocab_term(repetitions);

CREATE TABLE IF NOT EXISTS quiz_session (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    list_id         INTEGER NOT NULL REFERENCES vocab_list(id),
    mode            TEXT    NOT NULL,
    started_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    completed_at    TEXT,
    total_terms     INTEGER NOT NULL DEFAULT 0,
    correct_count   INTEGER NOT NULL DEFAULT 0,
    incorrect_count INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS quiz_result (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id      INTEGER NOT NULL REFERENCES quiz_session(id) ON DELETE CASCADE,
    term_id         INTEGER NOT NULL REFERENCES vocab_term(id),
    was_correct     INTEGER NOT NULL,
    answer_given    TEXT,
    grade           REAL,
    time_taken_ms   INTEGER,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_quiz_result_session ON quiz_result(session_id);

CREATE TABLE IF NOT EXISTS grammar_exercise (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    type            TEXT    NOT NULL,
    source_lang_id  INTEGER NOT NULL REFERENCES language(id),
    target_lang_id  INTEGER NOT NULL REFERENCES language(id),
    topic           TEXT,
    cefr_level      TEXT,
    difficulty      INTEGER NOT NULL DEFAULT 1,
    prompt          TEXT    NOT NULL,
    correct_answer  TEXT    NOT NULL,
    acceptable_answers TEXT,
    hint            TEXT,
    explanation     TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_grammar_type_topic ON grammar_exercise(type, topic);

CREATE TABLE IF NOT EXISTS exercise_session (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    completed_at    TEXT,
    total_exercises INTEGER NOT NULL DEFAULT 0,
    correct_count   INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS exercise_result (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id      INTEGER NOT NULL REFERENCES exercise_session(id) ON DELETE CASCADE,
    exercise_id     INTEGER NOT NULL REFERENCES grammar_exercise(id),
    was_correct     INTEGER NOT NULL,
    answer_given    TEXT,
    time_taken_ms   INTEGER,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS daily_activity (
    date            TEXT PRIMARY KEY,
    quiz_count      INTEGER NOT NULL DEFAULT 0,
    exercise_count  INTEGER NOT NULL DEFAULT 0,
    terms_reviewed  INTEGER NOT NULL DEFAULT 0,
    time_spent_min  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS user_setting (
    key             TEXT PRIMARY KEY,
    value           TEXT NOT NULL,
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT OR IGNORE INTO user_setting (key, value) VALUES
    ('source_lang',     'en'),
    ('target_lang',     'fr'),
    ('daily_goal',      '20'),
    ('default_quiz_mode','flashcard'),
    ('session_size',    '10'),
    ('db_version',      '1');

INSERT OR IGNORE INTO language (code, name, flag_emoji) VALUES
    ('en', 'English',     '🇬🇧'),
    ('fr', 'French',      '🇫🇷'),
    ('es', 'Spanish',     '🇪🇸'),
    ('de', 'German',      '🇩🇪'),
    ('it', 'Italian',     '🇮🇹'),
    ('pt', 'Portuguese',  '🇵🇹');
