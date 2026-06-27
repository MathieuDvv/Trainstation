from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import Screen
from textual.widgets import Static


class ListDetailScreen(Screen):
    """View and edit terms in a vocabulary list."""

    BINDINGS = [
        ("escape", "pop_screen", "Back"),
    ]

    def compose(self) -> ComposeResult:
        with Vertical(classes="placeholder-screen"):
            yield Static("List Detail", classes="placeholder-title")
            yield Static(
                "View and edit terms in this list.\n\n"
                "Scroll through terms, inline-edit fields.",
                classes="placeholder-body",
            )
            yield Static("Coming soon...", classes="placeholder-footer")
