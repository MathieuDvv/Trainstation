from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import Screen
from textual.widgets import Static


class ExerciseSetupScreen(Screen):
    """Select exercise type, topic, difficulty."""

    BINDINGS = [
        ("escape", "pop_screen", "Back"),
    ]

    def compose(self) -> ComposeResult:
        with Vertical(classes="placeholder-screen"):
            yield Static("Grammar Exercise Setup", classes="placeholder-title")
            yield Static(
                "Exercise types:\n"
                "· Fill in the blank\n"
                "· Conjugation drill\n"
                "· Sentence building\n"
                "· Error correction",
                classes="placeholder-body",
            )
            yield Static("Coming soon...", classes="placeholder-footer")
