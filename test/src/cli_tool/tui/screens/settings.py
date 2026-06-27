from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import Screen
from textual.widgets import Static


class SettingsScreen(Screen):
    """Configure language pair, daily goal, quiz preferences."""

    BINDINGS = [
        ("escape", "pop_screen", "Back"),
    ]

    def compose(self) -> ComposeResult:
        with Vertical(classes="placeholder-screen"):
            yield Static("Settings", classes="placeholder-title")
            yield Static(
                "Configure:\n"
                "· Language pair (source → target)\n"
                "· Daily goal\n"
                "· Default quiz mode\n"
                "· Session size\n"
                "· Data management",
                classes="placeholder-body",
            )
            yield Static("Coming soon...", classes="placeholder-footer")
