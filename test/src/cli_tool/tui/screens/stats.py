from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import Screen
from textual.widgets import Static


class StatsScreen(Screen):
    """Dashboard with streaks, mastery distribution, and activity."""

    BINDINGS = [
        ("escape", "pop_screen", "Back"),
    ]

    def compose(self) -> ComposeResult:
        with Vertical(classes="placeholder-screen"):
            yield Static("Progress & Stats", classes="placeholder-title")
            yield Static(
                "Dashboard: streaks, vocabulary mastery distribution,\n"
                "daily/weekly/monthly stats, per-list progress.",
                classes="placeholder-body",
            )
            yield Static("Coming soon...", classes="placeholder-footer")
