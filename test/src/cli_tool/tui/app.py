from textual.app import App
from textual.binding import Binding

from cli_tool.db.connection import close_db
from cli_tool.db.migrate import run_migrations
from cli_tool.tui.screens.exercise_results import ExerciseResultsScreen
from cli_tool.tui.screens.exercise_session import ExerciseSessionScreen
from cli_tool.tui.screens.exercise_setup import ExerciseSetupScreen
from cli_tool.tui.screens.help_screen import HelpScreen
from cli_tool.tui.screens.list_detail import ListDetailScreen
from cli_tool.tui.screens.main_menu import MainMenuScreen
from cli_tool.tui.screens.quiz_results import QuizResultsScreen
from cli_tool.tui.screens.quiz_session import QuizSessionScreen
from cli_tool.tui.screens.quiz_setup import QuizSetupScreen
from cli_tool.tui.screens.settings import SettingsScreen
from cli_tool.tui.screens.stats import StatsScreen
from cli_tool.tui.screens.word_lists import WordListsScreen
from cli_tool.tui.theme import CSS


class TrainApp(App):
    """Language learning TUI application — Trainstation."""

    CSS = CSS

    SCREENS = {
        "main_menu": MainMenuScreen,
        "quiz_setup": QuizSetupScreen,
        "quiz_session": QuizSessionScreen,
        "quiz_results": QuizResultsScreen,
        "exercise_setup": ExerciseSetupScreen,
        "exercise_session": ExerciseSessionScreen,
        "exercise_results": ExerciseResultsScreen,
        "word_lists": WordListsScreen,
        "list_detail": ListDetailScreen,
        "stats": StatsScreen,
        "settings": SettingsScreen,
        "help": HelpScreen,
    }

    BINDINGS = [
        Binding("ctrl+q", "quit", "Quit", priority=True),
        Binding("ctrl+h", "push_screen('help')", "Help"),
        Binding("question_mark", "push_screen('help')", "Help"),
    ]

    def on_mount(self) -> None:
        run_migrations()
        self.push_screen("main_menu")

    def on_unmount(self) -> None:
        close_db()
