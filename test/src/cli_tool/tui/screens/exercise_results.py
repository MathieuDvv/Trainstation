from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import Screen
from textual.widgets import Static


class ExerciseResultsScreen(Screen):
    """Exercise session summary."""

    BINDINGS = [
        ("escape", "pop_screen", "Back to menu"),
    ]

    def compose(self) -> ComposeResult:
        with Vertical(classes="placeholder-screen"):
            yield Static("Exercise Results", classes="placeholder-title")
            yield Static(
                "Session summary: accuracy, time, topics covered.",
                classes="placeholder-body",
            )
            yield Static("Coming soon...", classes="placeholder-footer")
