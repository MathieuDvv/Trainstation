from textual import work
from textual.app import App, ComposeResult
from textual.containers import VerticalScroll
from textual.widgets import Footer, Header, Input, Static

from .config import get_config
from .deepseek import (
    DeepseekAPIError,
    DeepseekAuthError,
    DeepseekClient,
    DeepseekConnectionError,
    DeepseekRateLimitError,
    DeepseekTimeoutError,
)


class ChatMessage(Static):
    """A widget for displaying a chat message."""

    pass


class ChatApp(App):
    """A basic chat TUI with Deepseek integration."""

    CSS = """
    #chat-history {
        height: 1fr;
        border: solid green;
    }
    #chat-input {
        dock: bottom;
        margin: 1;
    }
    .user-message {
        margin: 1 2;
        color: auto;
    }
    .assistant-message {
        margin: 1 2;
        color: auto;
    }
    .error-message {
        margin: 1 2;
        color: red;
    }
    """

    BINDINGS = [
        ("ctrl+q", "quit", "Quit"),
    ]

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.config = get_config()
        self.messages = []
        # Create client if configured
        if self.config.is_configured:
            self.client = DeepseekClient(api_key=self.config.api_key)
        else:
            self.client = None

    def compose(self) -> ComposeResult:
        yield Header()
        with VerticalScroll(id="chat-history"):
            pass
        yield Input(placeholder="Type a message and press Enter...", id="chat-input")
        yield Footer()

    def on_mount(self) -> None:
        self.query_one(Input).focus()
        history = self.query_one("#chat-history", VerticalScroll)
        if not self.client:
            history.mount(
                ChatMessage(
                    "Please set DEEPSEEK_API_KEY environment variable "
                    "or configure it via file to use the chat.",
                    classes="error-message",
                )
            )

    def on_input_submitted(self, event: Input.Submitted) -> None:
        if not self.client:
            return

        message = event.value.strip()
        if not message:
            return

        # Append user message
        history = self.query_one("#chat-history", VerticalScroll)
        history.mount(
            ChatMessage(
                f"[bold green]You:[/bold green] {message}", classes="user-message"
            )
        )
        self.query_one(Input).value = ""

        self.messages.append({"role": "user", "content": message})

        # Start assistant response
        msg_widget = ChatMessage(
            "[bold blue]Assistant:[/bold blue] ", classes="assistant-message"
        )
        history.mount(msg_widget)
        history.scroll_end(animate=False)

        self.call_api(msg_widget)

    @work(thread=True)
    def call_api(self, widget: ChatMessage) -> None:
        try:
            content = ""
            for chunk in self.client.chat_completion_stream(
                model=self.config.model,
                messages=self.messages,
                temperature=self.config.temperature,
                max_tokens=self.config.max_tokens,
            ):
                delta = chunk.choices[0].message.content
                if delta:
                    content += delta
                    # Update widget from thread
                    self.call_from_thread(
                        widget.update, f"[bold blue]Assistant:[/bold blue] {content}"
                    )
                    # Scroll to end
                    history = self.app.query_one("#chat-history", VerticalScroll)
                    self.call_from_thread(history.scroll_end, animate=False)

            # Save to history
            self.messages.append({"role": "assistant", "content": content})

        except DeepseekAuthError:
            self.call_from_thread(self._show_error, "Error: Invalid API key.")
        except DeepseekRateLimitError:
            self.call_from_thread(
                self._show_error, "Error: Rate limit exceeded. Try again later."
            )
        except DeepseekTimeoutError:
            self.call_from_thread(self._show_error, "Error: Request timed out.")
        except DeepseekConnectionError:
            self.call_from_thread(
                self._show_error, "Error: Could not connect to Deepseek API."
            )
        except DeepseekAPIError as exc:
            self.call_from_thread(self._show_error, f"Error ({exc.status_code}): {exc}")
        except Exception as exc:
            self.call_from_thread(self._show_error, f"Unexpected error: {exc}")

    def _show_error(self, message: str) -> None:
        history = self.query_one("#chat-history", VerticalScroll)
        history.mount(ChatMessage(message, classes="error-message"))
        history.scroll_end(animate=False)
