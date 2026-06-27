from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import ModalScreen
from textual.widgets import Static


class ImportDialog(ModalScreen[str | None]):
    """CSV/JSON import dialog with file path input."""

    BINDINGS = [
        ("escape", "pop_screen", "Cancel"),
    ]

    def compose(self) -> ComposeResult:
        with Vertical(id="dialog-container"):
            yield Static("Import", id="dialog-title")
            yield Static(
                "Select a file to import.\n\n"
                "Supported formats: CSV, JSON, Anki .apkg",
                id="dialog-message",
            )
