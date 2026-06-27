from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import Screen
from textual.widgets import Static


class QuizResultsScreen(Screen):
    """Session summary with stats after a quiz."""

    BINDINGS = [
        ("escape", "pop_screen", "Back to menu"),
        ("enter", "back_to_menu", "Main Menu"),
    ]

    def compose(self) -> ComposeResult:
        with Vertical(classes="placeholder-screen"):
            yield Static("Quiz Results", classes="placeholder-title")
            yield Static(
                "Accuracy, time-per-card, words to review, streak update.",
                classes="placeholder-body",
            )
            yield Static("Coming soon...", classes="placeholder-footer")

    def action_back_to_menu(self) -> None:
        self.app.pop_screen()
        self.app.pop_screen()
