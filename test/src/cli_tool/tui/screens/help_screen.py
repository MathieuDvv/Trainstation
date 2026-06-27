from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import ModalScreen
from textual.widgets import Static


class HelpScreen(ModalScreen):
    """Keyboard shortcuts overlay."""

    BINDINGS = [
        ("escape", "dismiss", "Close"),
        ("q", "dismiss", "Close"),
    ]

    def compose(self) -> ComposeResult:
        with Vertical(id="help-overlay"):
            yield Static("⌨  Keyboard Shortcuts", id="help-title")
            yield Static(
                "Navigation\n"
                "  1-5       — Jump to menu items from main screen\n"
                "  Escape    — Go back / dismiss popup\n"
                "  Ctrl+Q    — Quit from anywhere\n"
                "\n"
                "General\n"
                "  Ctrl+H / ? — Show this help overlay\n"
                "  Ctrl+S    — Quick-study (coming soon)\n"
                "\n"
                "Quiz (coming soon)\n"
                "  0-5       — SM-2 self-grade (flashcard mode)\n"
                "  Enter     — Submit answer / flip card\n"
                "  Tab       — Next option (multiple choice)",
                classes="help-section",
            )
