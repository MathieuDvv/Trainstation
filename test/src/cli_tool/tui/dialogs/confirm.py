from textual import on
from textual.app import ComposeResult
from textual.containers import Horizontal, Vertical
from textual.screen import ModalScreen
from textual.widgets import Button, Static


class ConfirmDialog(ModalScreen[bool]):
    """Generic yes/no confirmation dialog."""

    BINDINGS = [
        ("y", "confirm", "Yes"),
        ("n", "cancel", "No"),
        ("escape", "cancel", "No"),
    ]

    def __init__(self, title: str = "Confirm", message: str = "Are you sure?"):
        super().__init__()
        self._title = title
        self._message = message

    def compose(self) -> ComposeResult:
        with Vertical(id="dialog-container"):
            yield Static(self._title, id="dialog-title")
            yield Static(self._message, id="dialog-message")
            with Horizontal(id="dialog-buttons"):
                yield Button(
                    "Yes", variant="primary", id="btn-yes", classes="dialog-btn"
                )
                yield Button(
                    "No", variant="default", id="btn-no", classes="dialog-btn"
                )

    def action_confirm(self) -> None:
        self.dismiss(True)

    def action_cancel(self) -> None:
        self.dismiss(False)

    @on(Button.Pressed, "#btn-yes")
    def _on_yes(self) -> None:
        self.dismiss(True)

    @on(Button.Pressed, "#btn-no")
    def _on_no(self) -> None:
        self.dismiss(False)
