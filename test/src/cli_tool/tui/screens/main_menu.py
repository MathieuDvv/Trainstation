from textual import on
from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import Screen
from textual.widgets import Button, Static


class MainMenuScreen(Screen):
    """Main menu with shortcuts and daily summary."""

    BINDINGS = [
        ("1", "study", "Study"),
        ("2", "grammar", "Grammar"),
        ("3", "lists", "Word Lists"),
        ("4", "stats", "Stats"),
        ("5", "settings", "Settings"),
        ("ctrl+q", "quit", "Quit"),
        ("ctrl+h", "push_screen('help')", "Help"),
    ]

    # Key-chord aliases for Query.One to grab key 1-5 from the screen
    def compose(self) -> ComposeResult:
        with Vertical(id="main-menu"):
            yield Static("🚂  Trainstation", id="main-title")
            yield Static("Language Lab — Textual Edition", id="main-subtitle")

            with Vertical(id="menu-buttons"):
                yield Button("1. Study", id="btn-study", classes="menu-btn")
                yield Button(
                    "2. Grammar Exercises", id="btn-grammar", classes="menu-btn"
                )
                yield Button("3. Word Lists", id="btn-lists", classes="menu-btn")
                yield Button("4. Progress & Stats", id="btn-stats", classes="menu-btn")
                yield Button("5. Settings", id="btn-settings", classes="menu-btn")

            yield Static(
                "Today: 0 terms due · Streak: 0 days",
                id="main-summary",
            )

    def action_study(self) -> None:
        self.app.push_screen("quiz_setup")

    def action_grammar(self) -> None:
        self.app.push_screen("exercise_setup")

    def action_lists(self) -> None:
        self.app.push_screen("word_lists")

    def action_stats(self) -> None:
        self.app.push_screen("stats")

    def action_settings(self) -> None:
        self.app.push_screen("settings")

    @on(Button.Pressed, "#btn-study")
    def _on_study(self) -> None:
        self.app.push_screen("quiz_setup")

    @on(Button.Pressed, "#btn-grammar")
    def _on_grammar(self) -> None:
        self.app.push_screen("exercise_setup")

    @on(Button.Pressed, "#btn-lists")
    def _on_lists(self) -> None:
        self.app.push_screen("word_lists")

    @on(Button.Pressed, "#btn-stats")
    def _on_stats(self) -> None:
        self.app.push_screen("stats")

    @on(Button.Pressed, "#btn-settings")
    def _on_settings(self) -> None:
        self.app.push_screen("settings")
