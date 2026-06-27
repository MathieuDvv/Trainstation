from dataclasses import dataclass, field
from datetime import date
from enum import Enum


class QuizMode(Enum):
    FLASHCARD = "flashcard"
    MULTIPLE_CHOICE = "multiple_choice"
    TYPING = "typing"


class ExerciseType(Enum):
    FILL_BLANK = "fill_blank"
    CONJUGATION = "conjugation"
    SENTENCE_BUILD = "sentence_build"
    ERROR_CORRECT = "error_correct"


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
    source_lang_id: int
    target_lang_id: int
    description: str | None = None
    topic: str | None = None
    cefr_level: str | None = None
    is_builtin: bool = False
    term_count: int = 0
    due_count: int = 0
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
    grade: float | None = None
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
    date: str
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
