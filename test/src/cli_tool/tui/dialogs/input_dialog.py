from textual import on
from textual.app import ComposeResult
from textual.containers import Horizontal, Vertical
from textual.screen import ModalScreen
from textual.widgets import Button, Input, Static


class InputDialog(ModalScreen[str | None]):
    """Single-line text input modal."""

    BINDINGS = [
        ("escape", "cancel", "Cancel"),
    ]

    def __init__(self, title: str = "Input", prompt: str = "Enter value:"):
        super().__init__()
        self._title = title
        self._prompt_text = prompt

    def compose(self) -> ComposeResult:
        with Vertical(id="dialog-container"):
            yield Static(self._title, id="dialog-title")
            yield Static(self._prompt_text, id="dialog-message")
            yield Input(id="dialog-input")
            with Horizontal(id="dialog-buttons"):
                yield Button(
                    "OK", variant="primary", id="btn-ok", classes="dialog-btn"
                )
                yield Button(
                    "Cancel", variant="default", id="btn-cancel", classes="dialog-btn"
                )

    def on_mount(self) -> None:
        self.query_one("#dialog-input", Input).focus()

    @on(Input.Submitted, "#dialog-input")
    def _on_submit(self) -> None:
        value = self.query_one("#dialog-input", Input).value.strip()
        self.dismiss(value if value else None)

    def action_cancel(self) -> None:
        self.dismiss(None)

    @on(Button.Pressed, "#btn-ok")
    def _on_ok(self) -> None:
        value = self.query_one("#dialog-input", Input).value.strip()
        self.dismiss(value if value else None)

    @on(Button.Pressed, "#btn-cancel")
    def _on_cancel(self) -> None:
        self.dismiss(None)
