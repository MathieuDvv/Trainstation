from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import Screen
from textual.widgets import Static


class QuizSetupScreen(Screen):
    """Select list, mode, and session size before starting a quiz."""

    BINDINGS = [
        ("escape", "pop_screen", "Back"),
    ]

    def compose(self) -> ComposeResult:
        with Vertical(classes="placeholder-screen"):
            yield Static("Quiz Setup", classes="placeholder-title")
            yield Static(
                "Select a vocabulary list and quiz mode.\n\n"
                "Modes: Flashcard · Multiple Choice · Typing",
                classes="placeholder-body",
            )
            yield Static("Coming soon...", classes="placeholder-footer")
