from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import Screen
from textual.widgets import Static


class QuizSessionScreen(Screen):
    """Active quiz loop — flashcard, multiple choice, or typing mode."""

    BINDINGS = [
        ("escape", "pop_screen", "Back to setup"),
    ]

    def compose(self) -> ComposeResult:
        with Vertical(classes="placeholder-screen"):
            yield Static("Quiz Session", classes="placeholder-title")
            yield Static(
                "Answer the flashcard, multiple choice, or typing prompts.\n\n"
                "Graded with SM-2 spaced repetition.",
                classes="placeholder-body",
            )
            yield Static("Coming soon...", classes="placeholder-footer")
