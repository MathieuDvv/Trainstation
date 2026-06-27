from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import Screen
from textual.widgets import Static


class ExerciseSessionScreen(Screen):
    """Active exercise loop."""

    BINDINGS = [
        ("escape", "pop_screen", "Back to setup"),
    ]

    def compose(self) -> ComposeResult:
        with Vertical(classes="placeholder-screen"):
            yield Static("Grammar Exercise Session", classes="placeholder-title")
            yield Static(
                "Complete grammar exercises.\n\n"
                "Answer fill-in-the-blank, conjugation, sentence building,\n"
                "or error correction prompts.",
                classes="placeholder-body",
            )
            yield Static("Coming soon...", classes="placeholder-footer")
