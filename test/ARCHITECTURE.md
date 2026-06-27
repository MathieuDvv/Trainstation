# Language Learning TUI — Architecture Blueprint

> **Project codename:** Trainstation `/test` — repurposed from `cli-tool` chat client
> into a **language learning terminal application**.

---

## 1. Feature Specification

### 1.1 Vocabulary Quiz

The core learning loop. Users are presented with a word or definition and must
respond with the translation, select the correct answer from multiple choices,
or self-assess their recall (flashcard model).

| Sub-feature  | Description |
|-------------|-------------|
| **Flashcard mode**  | Display the *source* word; user mentally recalls the *target* translation, then reveals. Self-graded on a 0–5 scale (SM‑2 algorithm). |
| **Multiple choice** | Show the source word + 4 options (1 correct, 3 plausible distractors). Timed or untimed. |
| **Typing mode**  | Show the source word; user types the translation. Levenshtein-tolerant matching with near-miss feedback. |
| **Spaced repetition** | SM‑2 scheduling: each term has `easiness`, `interval`, `repetitions`, and `next_review`. Only due terms appear in a session. |
| **Word lists** | Quizzes are scoped to a *list* (e.g. "A1 Essentials", "Irregular Verbs", "Food"). Lists are taggable by topic and CEFR level. |
| **Session summary** | After a quiz, display accuracy, time-per-card, words to review, and streak update. |

### 1.2 Grammar Exercises

Interactive drills targeting structural competence in the target language.

| Sub-feature  | Description |
|-------------|-------------|
| **Fill-in-the-blank** | A sentence with one gap. User types or selects the missing word (conjugated verb, preposition, article, etc.). |
| **Conjugation drill** | Given an infinitive + person/number/tense, user types the correct conjugated form. |
| **Sentence building** | Given a set of words/tokens in random order, user assembles the correct sentence (drag-to-order or numbered tokens). |
| **Error correction** | A sentence with one deliberate error. User identifies or rewrites the correct form. |
| **Progressive difficulty** | Exercises are tagged with CEFR level and topic. The app tracks per-topic accuracy and suggests the next appropriate difficulty. |

### 1.3 Progress & Stats

| Sub-feature  | Description |
|-------------|-------------|
| **Dashboard** | Daily/weekly/monthly stats: quizzes completed, accuracy, time spent, terms learned. |
| **Streaks** | Consecutive days with at least one completed session. Fire emoji encouragement. |
| **Vocabulary mastery** | Distribution of terms across SM‑2 buckets (new → learning → mature). |
| **Per-list progress** | For each vocabulary list: % mastered, % due today, last studied. |
| **Export** | Dump stats and term data as CSV/JSON for external analysis. |

### 1.4 Word List Management

| Sub-feature  | Description |
|-------------|-------------|
| **Browse** | Table view of all lists with name, term count, language pair, topic. |
| **View/Edit** | Open a list tab, scroll through terms, inline-edit fields. |
| **Import** | CSV or JSON import with field mapping. Deduplication on source word. |
| **Export** | Export list as CSV/JSON/Anki-compatible format. |
| **Pre-built packs** | Ship bundled vocabulary packs for common language pairs (EN→FR, EN→ES, EN→DE). |

### 1.5 Settings

| Sub-feature  | Description |
|-------------|-------------|
| **Language pair** | Source and target language (`en→fr`, `en→es`, etc.) |
| **Daily goal** | Target number of terms to review per day. |
| **Quiz preferences** | Default quiz mode, session size, default time-per-card. |
| **Data management** | Reset progress, export all data, clear a list's history. |
| **Appearance** | Terminal theme colours if needed (respect terminal theme by default). |

---

## 2. Navigation Flow

```
┌─────────────────────────────────────────────┐
│                  App Launch                  │
│  (init DB, load config, check first run)     │
└──────────────────────┬──────────────────────┘
                       ▼
┌─────────────────────────────────────────────┐
│              Main Menu Screen               │
│  ┌──────────────────────────────────────┐   │
│  │  🚂  Trainstation — Language Lab     │   │
│  │                                      │   │
│  │  [1] Study (start a quiz session)   │   │
│  │  [2] Grammar Exercises              │   │
│  │  [3] Word Lists                     │   │
│  │  [4] Progress & Stats               │   │
│  │  [5] Settings                       │   │
│  │                                      │   │
│  │  Today: 12 terms due · Streak: 7 🔥  │   │
│  └──────────────────────────────────────┘   │
└───────┬─────┬─────┬─────┬─────┬───────────┘
        │     │     │     │     │
        ▼     ▼     ▼     ▼     ▼
   ┌────────┐┌──────┐┌──────┐┌──────┐┌──────┐
   │ Study  ││Grammar││Word  ││Stats ││Settings│
   │ Screen ││Screen││Lists ││Screen││Screen │
   └───┬────┘└──┬───┘└──┬───┘└──┬───┘└──┬────┘
       │        │       │       │       │
       ▼        ▼       ▼       ▼       ▼
   ┌────────┐┌──────┐┌──────┐
   │  Quiz  ││Exercise││List  │
   │Session ││Session││Detail│
   └───┬────┘└──┬───┘└──────┘
       │        │
       ▼        ▼
   ┌────────┐┌──────┐
   │Results ││Results│
   │ Screen ││Screen│
   └───┬────┘└──┬───┘
       │        │
       └────┬───┘
            ▼
     Back to Main Menu
```

**Key navigation rules:**

1. **Screen stack:** Textual's `Screen` system provides a natural back-stack.
   Pushing a screen returns to the previous one on dismiss/`Escape`.
2. **Keyboard-driven:** Everything accessible via keyboard. Mouse support as a
   secondary input for terminal emulators that support it.
3. **Global shortcuts:**
   - `Ctrl+Q` — Quit from anywhere
   - `Escape` — Go back / dismiss popup
   - `Ctrl+H` / `?` — Help overlay
   - `Ctrl+S` — Quick-study (jump directly to quiz from anywhere)
   - `1`–`5` — Menu item selection on Main Menu
4. **Modal dialogs** (Textual `ModalScreen`) for confirmations (quit, reset
   progress), and input prompts (import file path).
5. **No deep nesting:** Maximum 2–3 screens deep. Deep flows like quiz→results
   return directly to Main Menu.

---

## 3. Data Models

### 3.1 Storage Engine: SQLite via stdlib `sqlite3`

**Rationale against alternatives:**

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **sqlite3 (stdlib)** | Zero dependencies, single-file DB, transactional, supports WAL mode, sufficient for single-user TUI | Manual SQL (no ORM sugar) | **Chosen** |
| SQLAlchemy | ORM convenience, migrations (Alembic) | Heavy dependency for a TUI app, overkill for this schema | Not worth it |
| JSON/YAML files | Human-readable, trivial to edit | No query support, concurrent writes risky, no indexing, will degrade with 10k+ terms | Only for config |
| Pydantic + JSON | Type-safe serialization | Same file-based drawbacks; Pydantic adds a heavy dep | Not needed |
| Peewee | Lightweight ORM | Extra dependency for marginal benefit over raw sqlite3 | Optional future upgrade |

**schema.sql:**

```sql
-- Enable WAL mode for better concurrent read/write
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

-- =========================================================================
-- Languages
-- =========================================================================
CREATE TABLE IF NOT EXISTS language (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    code        TEXT    NOT NULL UNIQUE,  -- ISO 639-1: 'en', 'fr', 'de'
    name        TEXT    NOT NULL,         -- 'English', 'French', 'German'
    flag_emoji  TEXT                      -- 🇬🇧 🇫🇷 🇩🇪 (optional)
);

-- =========================================================================
-- Vocabulary Lists
-- =========================================================================
CREATE TABLE IF NOT EXISTS vocab_list (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT    NOT NULL,
    description     TEXT,
    source_lang_id  INTEGER NOT NULL REFERENCES language(id),
    target_lang_id  INTEGER NOT NULL REFERENCES language(id),
    topic           TEXT,               -- 'food', 'travel', 'business'
    cefr_level      TEXT,               -- 'A1', 'A2', 'B1', 'B2', 'C1', 'C2'
    is_builtin      INTEGER DEFAULT 0,  -- 1 = pre-bundled, cannot delete
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- =========================================================================
-- Vocabulary Terms
-- =========================================================================
CREATE TABLE IF NOT EXISTS vocab_term (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    list_id         INTEGER NOT NULL REFERENCES vocab_list(id) ON DELETE CASCADE,
    source_word     TEXT    NOT NULL,
    target_word     TEXT    NOT NULL,
    -- Optional metadata
    part_of_speech  TEXT,               -- 'noun', 'verb', 'adj', 'adv', etc.
    gender          TEXT,               -- 'm', 'f', 'n' (if applicable)
    example_sentence TEXT,
    pronunciation   TEXT,               -- IPA or phonetic hint
    -- SM-2 spaced repetition fields
    easiness_factor REAL    NOT NULL DEFAULT 2.5,   -- SM-2 EF (min 1.3)
    interval_days   INTEGER NOT NULL DEFAULT 0,     -- days until next review
    repetitions     INTEGER NOT NULL DEFAULT 0,     -- consecutive correct reps
    next_review     TEXT    NOT NULL DEFAULT (date('now')),  -- date of next review
    -- Bookkeeping
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    last_reviewed   TEXT,
    -- A term is unique within its list (same source word)
    UNIQUE(list_id, source_word)
);

CREATE INDEX IF NOT EXISTS idx_vocab_term_list     ON vocab_term(list_id);
CREATE INDEX IF NOT EXISTS idx_vocab_term_due      ON vocab_term(next_review, list_id);
CREATE INDEX IF NOT EXISTS idx_vocab_term_mastery  ON vocab_term(repetitions);

-- =========================================================================
-- Quiz Sessions
-- =========================================================================
CREATE TABLE IF NOT EXISTS quiz_session (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    list_id         INTEGER NOT NULL REFERENCES vocab_list(id),
    mode            TEXT    NOT NULL,  -- 'flashcard', 'multiple_choice', 'typing'
    started_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    completed_at    TEXT,
    total_terms     INTEGER NOT NULL DEFAULT 0,
    correct_count   INTEGER NOT NULL DEFAULT 0,
    incorrect_count INTEGER NOT NULL DEFAULT 0
);

-- =========================================================================
-- Quiz Results (per-term grading within a session)
-- =========================================================================
CREATE TABLE IF NOT EXISTS quiz_result (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id      INTEGER NOT NULL REFERENCES quiz_session(id) ON DELETE CASCADE,
    term_id         INTEGER NOT NULL REFERENCES vocab_term(id),
    was_correct     INTEGER NOT NULL,  -- 0 or 1
    answer_given    TEXT,
    grade           REAL,              -- SM-2 grade: 0–5 (NULL for non-flashcard)
    time_taken_ms   INTEGER,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_quiz_result_session ON quiz_result(session_id);

-- =========================================================================
-- Grammar Exercises (static content)
-- =========================================================================
CREATE TABLE IF NOT EXISTS grammar_exercise (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    type            TEXT    NOT NULL,  -- 'fill_blank', 'conjugation', 'sentence_build', 'error_correct'
    source_lang_id  INTEGER NOT NULL REFERENCES language(id),
    target_lang_id  INTEGER NOT NULL REFERENCES language(id),
    topic           TEXT,              -- 'past_tense', 'articles', 'prepositions'
    cefr_level      TEXT,
    difficulty      INTEGER NOT NULL DEFAULT 1,  -- 1 (easy) to 5 (hard)
    prompt          TEXT    NOT NULL,  -- The exercise text (may contain marker for gap)
    correct_answer  TEXT    NOT NULL,
    acceptable_answers TEXT,           -- JSON array of alternative correct answers
    hint            TEXT,
    explanation     TEXT,              -- Shown after answer: why it's correct
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_grammar_type_topic ON grammar_exercise(type, topic);

-- =========================================================================
-- Grammar Exercise Sessions
-- =========================================================================
CREATE TABLE IF NOT EXISTS exercise_session (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    completed_at    TEXT,
    total_exercises INTEGER NOT NULL DEFAULT 0,
    correct_count   INTEGER NOT NULL DEFAULT 0
);

-- =========================================================================
-- Exercise Results
-- =========================================================================
CREATE TABLE IF NOT EXISTS exercise_result (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id      INTEGER NOT NULL REFERENCES exercise_session(id) ON DELETE CASCADE,
    exercise_id     INTEGER NOT NULL REFERENCES grammar_exercise(id),
    was_correct     INTEGER NOT NULL,
    answer_given    TEXT,
    time_taken_ms   INTEGER,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

-- =========================================================================
-- Daily Activity Log (for streaks and stats)
-- =========================================================================
CREATE TABLE IF NOT EXISTS daily_activity (
    date            TEXT PRIMARY KEY,  -- 'YYYY-MM-DD'
    quiz_count      INTEGER NOT NULL DEFAULT 0,
    exercise_count  INTEGER NOT NULL DEFAULT 0,
    terms_reviewed  INTEGER NOT NULL DEFAULT 0,
    time_spent_min  INTEGER NOT NULL DEFAULT 0
);

-- =========================================================================
-- User Settings
-- =========================================================================
CREATE TABLE IF NOT EXISTS user_setting (
    key             TEXT PRIMARY KEY,
    value           TEXT NOT NULL,
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Seed default settings
INSERT OR IGNORE INTO user_setting (key, value) VALUES
    ('source_lang',     'en'),
    ('target_lang',     'fr'),
    ('daily_goal',      '20'),
    ('default_quiz_mode','flashcard'),
    ('session_size',    '10'),
    ('db_version',      '1');

-- Seed languages
INSERT OR IGNORE INTO language (code, name, flag_emoji) VALUES
    ('en', 'English',     '🇬🇧'),
    ('fr', 'French',      '🇫🇷'),
    ('es', 'Spanish',     '🇪🇸'),
    ('de', 'German',      '🇩🇪'),
    ('it', 'Italian',     '🇮🇹'),
    ('pt', 'Portuguese',  '🇵🇹');
```

### 3.2 Python Dataclass Layer

The dataclasses mirror the SQL schema and are used for in-memory representation
within the TUI. A lightweight repository pattern maps between the two.

```python
from dataclasses import dataclass, field
from datetime import date, datetime
from enum import Enum


# ---------- Enums ----------

class QuizMode(Enum):
    FLASHCARD = "flashcard"
    MULTIPLE_CHOICE = "multiple_choice"
    TYPING = "typing"

class ExerciseType(Enum):
    FILL_BLANK = "fill_blank"
    CONJUGATION = "conjugation"
    SENTENCE_BUILD = "sentence_build"
    ERROR_CORRECT = "error_correct"


# ---------- Core models ----------

@dataclass
class Language:
    id: int
    code: str
    name: str
    flag_emoji: str | None = None

@dataclass
class VocabList:
    id: int
    name: str
    description: str | None
    source_lang_id: int
    target_lang_id: int
    topic: str | None = None
    cefr_level: str | None = None
    is_builtin: bool = False
    term_count: int = 0          # populated at query time
    due_count: int = 0           # populated at query time
    created_at: str = ""
    updated_at: str = ""

@dataclass
class VocabTerm:
    id: int
    list_id: int
    source_word: str
    target_word: str
    part_of_speech: str | None = None
    gender: str | None = None
    example_sentence: str | None = None
    pronunciation: str | None = None
    easiness_factor: float = 2.5
    interval_days: int = 0
    repetitions: int = 0
    next_review: str = date.today().isoformat()
    created_at: str = ""
    last_reviewed: str | None = None

    @property
    def is_due(self) -> bool:
        return self.next_review <= date.today().isoformat()

    @property
    def mastery_level(self) -> str:
        """Categorize based on SM-2 repetitions."""
        if self.repetitions == 0:
            return "new"
        if self.repetitions < 3:
            return "learning"
        if self.repetitions < 5:
            return "reviewing"
        return "mastered"

@dataclass
class QuizSession:
    id: int
    list_id: int
    mode: str
    started_at: str = ""
    completed_at: str | None = None
    total_terms: int = 0
    correct_count: int = 0
    incorrect_count: int = 0

@dataclass
class QuizResult:
    id: int
    session_id: int
    term_id: int
    was_correct: bool
    answer_given: str | None = None
    grade: float | None = None       # SM-2: 0–5
    time_taken_ms: int | None = None

@dataclass
class GrammarExercise:
    id: int
    type: str
    source_lang_id: int
    target_lang_id: int
    prompt: str
    correct_answer: str
    topic: str | None = None
    cefr_level: str | None = None
    difficulty: int = 1
    acceptable_answers: list[str] = field(default_factory=list)
    hint: str | None = None
    explanation: str | None = None

@dataclass
class ExerciseSession:
    id: int
    started_at: str = ""
    completed_at: str | None = None
    total_exercises: int = 0
    correct_count: int = 0

@dataclass
class ExerciseResult:
    id: int
    session_id: int
    exercise_id: int
    was_correct: bool
    answer_given: str | None = None
    time_taken_ms: int | None = None

@dataclass
class DailyActivity:
    date: str                      # 'YYYY-MM-DD'
    quiz_count: int = 0
    exercise_count: int = 0
    terms_reviewed: int = 0
    time_spent_min: int = 0

@dataclass
class UserSettings:
    source_lang: str = "en"
    target_lang: str = "fr"
    daily_goal: int = 20
    default_quiz_mode: str = "flashcard"
    session_size: int = 10
```

---

## 4. Python Library Selection

| Library | Version   | Purpose | Justification |
|---------|-----------|---------|---------------|
| **textual** | `>=0.40.0` | TUI framework | ✅ Already in dependencies. Most mature Python TUI framework. Built-in `Screen` stack for navigation, CSS theming, `@work` decorator for background threads, reactive attributes, and `Footer`/`Header` widgets. |
| **click** | `>=8.1` | CLI entry points | ✅ Already in deps. `cli-tool study`, `cli-tool stats`, `cli-tool import`, etc. |
| **rich** | *(implicit via textual)* | Fallback console output | Textual bundles Rich. Use for CLI-only commands where a full TUI is overkill (e.g. `cli-tool export --csv`). |
| **httpx** | `>=0.25` | HTTP client | ✅ Already in deps. Reserved for future features (online dictionary lookup, cloud sync), not needed for v1. |
| **pyyaml** | `>=6` | Config files | ✅ Already in deps. For `~/.trainstation/config.yaml` (API keys, preferences that don't belong in SQLite). |
| **sqlite3** | *(stdlib)* | Database | Zero extra dependencies. SQLite is perfect for a single-user TUI app. WAL mode gives concurrent read/write. |

**Notable exclusions and why:**

| Library | Why NOT included |
|---------|-----------------|
| SQLAlchemy / Peewee | Overkill for a 10-table schema. Add if the schema or query complexity grows significantly. |
| Pydantic | Heavy dep. Dataclasses + manual validation is sufficient and has no runtime cost. |
| Alembic | Schema migrations managed with a simple version integer in `user_setting` + manual migration SQL files. For v1, just recreate. |
| Rich (standalone) | Textual already bundles Rich. If CLI-only reports are needed, the `rich` console API can be used with `from rich.console import Console` (it's already installed as a textual dependency). |
| pytest | ✅ Already in dev deps. Used for testing. |

---

## 5. Source Tree Layout

```
src/
└── cli_tool/
    ├── __init__.py
    ├── __main__.py              # python -m cli_tool
    ├── cli.py                   # Click CLI group (main entry)
    ├── config.py                # Config dataclass + YAML loading (keep as-is, extend)
    │
    ├── db/
    │   ├── __init__.py
    │   ├── schema.sql           # DDL (the full schema above)
    │   ├── connection.py        # get_db() — singleton connection with WAL pragma
    │   ├── migrate.py           # Versioned migration runner
    │   └── seed.py              # Seed built-in vocab lists + grammar exercises
    │
    ├── models/
    │   ├── __init__.py          # Re-export all dataclasses
    │   └── models.py            # All dataclasses + enums
    │
    ├── repository/
    │   ├── __init__.py
    │   ├── vocab.py             # VocabListRepo, VocabTermRepo
    │   ├── quiz.py              # QuizSessionRepo, QuizResultRepo
    │   ├── grammar.py           # GrammarExerciseRepo, ExerciseSessionRepo
    │   └── stats.py             # DailyActivityRepo, aggregate queries
    │
    ├── services/
    │   ├── __init__.py
    │   ├── sm2.py               # SM‑2 algorithm implementation
    │   ├── quiz_service.py      # Orchestrates quiz sessions
    │   ├── exercise_service.py  # Orchestrates exercise sessions
    │   ├── stats_service.py     # Computes streaks, mastery, aggregates
    │   └── import_service.py    # CSV/JSON/Anki import logic
    │
    ├── tui/
    │   ├── __init__.py
    │   ├── app.py               # TrainApp(App) — main Textual app, screen management
    │   ├── theme.py             # CSS variables, colour palette
    │   ├── widgets.py           # Reusable widgets (ProgressBar, TermCard, etc.)
    │   │
    │   ├── screens/
    │   │   ├── __init__.py
    │   │   ├── main_menu.py     # Main menu with shortcuts and daily summary
    │   │   ├── quiz_setup.py    # Select list, mode, session size
    │   │   ├── quiz_session.py  # Active quiz loop (flashcard / MC / typing)
    │   │   ├── quiz_results.py  # Session summary with stats
    │   │   ├── exercise_setup.py# Select exercise type, topic, difficulty
    │   │   ├── exercise_session.py # Active exercise loop
    │   │   ├── exercise_results.py # Exercise session summary
    │   │   ├── word_lists.py    # Browse, manage lists
    │   │   ├── list_detail.py   # View/edit terms in a list
    │   │   ├── stats.py         # Dashboard, streaks, mastery distribution
    │   │   ├── settings.py      # Settings screen
    │   │   └── help.py          # Keyboard shortcuts overlay
    │   │
    │   └── dialogs/
    │       ├── __init__.py
    │       ├── confirm.py       # Generic yes/no confirmation
    │       ├── input_dialog.py  # Single-line text input prompt
    │       └── import_dialog.py # CSV import: file picker + field mapping
    │
    ├── deepseek/                # (keep existing, dormant for v1)
    │   └── ...
    │
    └── export/
        ├── __init__.py
        └── exporters.py         # CSV, JSON, Anki .apkg export
```

---

## 6. Key Architecture Decisions

### 6.1 Screen-per-feature pattern

Each major feature gets its own Textual `Screen`. Screens are pushed onto the
Textual screen stack, providing built-in back-navigation. This keeps the app
tree shallow — the `TrainApp` class delegates to sub-screens.

```python
# app.py
class TrainApp(App):
    SCREENS = {
        "main_menu": MainMenuScreen,
        "quiz_setup": QuizSetupScreen,
        "quiz_session": QuizSessionScreen,
        "quiz_results": QuizResultsScreen,
        # ... etc
    }

    def on_mount(self):
        self.push_screen("main_menu")
```

### 6.2 SM-2 Spaced Repetition

The [SM-2 algorithm](https://www.supermemo.com/en/archives1990-2015/english/ol/sm2)
is the industry standard for flashcard scheduling. Implementation (`services/sm2.py`):

```python
def sm2(quality: int, repetitions: int, ef: float, interval: int) -> tuple[float, int, int]:
    """
    quality: 0–5 (user self-assessment)
        0 = complete blackout
        5 = perfect response

    Returns: (new_ef, new_repetitions, new_interval_days)
    """
    if quality >= 3:
        if repetitions == 0:
            interval = 1
        elif repetitions == 1:
            interval = 6
        else:
            interval = round(interval * ef)
        repetitions += 1
    else:
        repetitions = 0
        interval = 1

    ef = ef + (0.1 - (5 - quality) * (0.08 + (5 - quality) * 0.02))
    ef = max(1.3, ef)

    return ef, repetitions, interval
```

This is called after each flashcard response. `next_review` is set to `today + interval` days.

### 6.3 Database access pattern

- **Single connection, WAL mode.** Opened once at `TrainApp.on_mount()`, stored as a module-level singleton via `db/connection.py`.
- **Repository classes** wrap raw SQL queries. They accept a `connection` parameter (dependency injection for testability).
- **No async.** Textual's `@work(thread=True)` decorator handles background work without blocking the UI. Since SQLite is thread-safe with a single connection in WAL mode, reads can happen on the main thread and writes in workers.
- **Migrations** run on startup. Version stored in `user_setting`. Each migration is a numbered `.sql` file in `db/migrations/`.

### 6.4 Quiz session state machine

```
         ┌─────────────────────┐
         │   SETUP              │  User picks list + mode + size
         └─────────┬───────────┘
                   ▼
         ┌─────────────────────┐
    ┌───▶│   INTRO              │  "10 terms from A1 Essentials. Ready?"
    │    └─────────┬───────────┘
    │              ▼
    │    ┌─────────────────────┐
    │    │   SHOW_TERM          │  Display source word
    │    └─────────┬───────────┘
    │              ▼
    │    ┌─────────────────────┐
    │    │   WAIT_ANSWER        │  User responds (types, selects, or reveals)
    │    └─────────┬───────────┘
    │              ▼
    │    ┌─────────────────────┐
    │    │   GRADE (flashcard)  │  If flashcard mode: 0–5 self-grade
    │    │ or                   │
    │    │   CHECK_ANSWER       │  If MC/typing: auto-check
    │    └─────────┬───────────┘
    │              ▼
    │    ┌─────────────────────┐
    │    │   FEEDBACK           │  Show correct answer + example sentence
    │    └─────────┬───────────┘
    │              ▼
    │         [more terms?]
    │      yes /       \ no
    │      /              \
    │     ▼                ▼
    └────┘          ┌─────────────────────┐
                    │   RESULTS            │  Session summary
                    └─────────┬───────────┘
                              ▼
                         Back to Menu
```

### 6.5 Distractor generation for multiple-choice

For the multiple-choice quiz mode, distractors (wrong answers) are selected
from other terms in the same vocabulary list to ensure similarity in topic and
difficulty. This avoids obviously-wrong distractors and tests real
discrimination.

Algorithm:
1. Pick the correct answer.
2. From the same list, select 3 random terms whose `target_word` differs from
   the correct one.
3. Shuffle all 4 options.

---

## 7. Testing Strategy

| Layer | Tool | What to test |
|-------|------|-------------|
| **Unit** | pytest | `sm2()` algorithm, repository queries (with `:memory:` SQLite), distractor selection |
| **Integration** | pytest | Full quiz flow: create session → submit answers → verify SM‑2 update → verify daily_activity |
| **TUI** | textual-testing / pytest | Screen navigation, widget rendering, keyboard input handling (snapshot or programmatic) |

---

## 8. Implementation Phases

### Phase 1 — Foundation (skeleton + DB + data entry)
- [ ] Rewrite `cli_tool` entry to launch language TUI instead of chat
- [ ] Implement `db/schema.sql`, `db/connection.py`, `db/migrate.py`, `db/seed.py`
- [ ] Implement `models/models.py` (all dataclasses)
- [ ] Seed one language pair (en→fr) with 200–300 A1/A2 vocabulary terms
- [ ] Seed 50 grammar exercises (all 4 types)
- [ ] Implement all repository classes

### Phase 2 — Quiz core loop
- [ ] `TrainApp` with main menu screen
- [ ] `quiz_setup` screen (list picker, mode picker, session size)
- [ ] `quiz_session` screen: flashcard mode with SM‑2 grading
- [ ] `quiz_results` screen
- [ ] `services/sm2.py`
- [ ] `services/quiz_service.py`
- [ ] Working end-to-end quiz flow

### Phase 3 — Grammar + MC + typing
- [ ] `exercise_setup` and `exercise_session` screens
- [ ] Multiple-choice quiz mode with distractor generation
- [ ] Typing quiz mode with Levenshtein-tolerant matching
- [ ] `services/exercise_service.py`

### Phase 4 — Stats, lists, polish
- [ ] `stats` screen (streaks, mastery, activity)
- [ ] `word_lists` and `list_detail` screens
- [ ] CSV/JSON import/export
- [ ] Settings screen
- [ ] Help overlay

### Phase 5 — Extras (v2)
- [ ] Anki `.apkg` export
- [ ] Online dictionary lookup (httpx-based)
- [ ] Additional language packs
- [ ] More grammar exercise types
