from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import Screen
from textual.widgets import Static


class WordListsScreen(Screen):
    """Browse and manage vocabulary lists."""

    BINDINGS = [
        ("escape", "pop_screen", "Back"),
    ]

    def compose(self) -> ComposeResult:
        with Vertical(classes="placeholder-screen"):
            yield Static("Word Lists", classes="placeholder-title")
            yield Static(
                "Browse, view, edit, import, and export vocabulary lists.\n\n"
                "Features: CSV/JSON import, Anki export.",
                classes="placeholder-body",
            )
            yield Static("Coming soon...", classes="placeholder-footer")
