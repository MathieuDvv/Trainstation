import time
from textual.app import App, ComposeResult
from textual.containers import VerticalScroll
from textual.widgets import Header, Footer, Input, Static
from textual import work

class ChatApp(App):
    CSS = """
    #chat-history {
        height: 1fr;
        border: solid green;
    }
    #chat-input {
        dock: bottom;
        margin: 1;
    }
    .message {
        margin: 1;
    }
    """

    def compose(self) -> ComposeResult:
        yield Header()
        with VerticalScroll(id="chat-history"):
            pass
        yield Input(id="chat-input")
        yield Footer()

    def on_input_submitted(self, event: Input.Submitted) -> None:
        message = event.value.strip()
        if message:
            history = self.query_one("#chat-history", VerticalScroll)
            history.mount(Static(f"You: {message}", classes="message"))
            self.query_one(Input).value = ""
            
            # Start assistant response
            msg_widget = Static("Assistant: ", classes="message")
            history.mount(msg_widget)
            history.scroll_end(animate=False)
            self.stream_response(msg_widget)

    @work(thread=True)
    def stream_response(self, widget: Static):
        content = "Assistant: "
        for i in range(5):
            time.sleep(0.5)
            content += f"chunk {i} "
            self.call_from_thread(widget.update, content)
            self.call_from_thread(self.query_one("#chat-history", VerticalScroll).scroll_end, animate=False)

if __name__ == "__main__":
    app = ChatApp()
    # app.run() # we just syntax check this
