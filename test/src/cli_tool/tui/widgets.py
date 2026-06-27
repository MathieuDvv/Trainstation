from textual.widgets import Button, Static


class MenuButton(Button):
    """A styled button for the main menu."""

    DEFAULT_CSS = """
    MenuButton {
        width: 100%;
        margin-bottom: 1;
    }
    """


class PlaceholderScreen(Static):
    """A placeholder screen shown for unimplemented features."""

    def __init__(self, title: str, description: str = "Coming soon..."):
        super().__init__(
            f"[bold $secondary]{title}[/]\n\n[dim]{description}[/]\n\n"
            "[$success]Press Escape to return to Main Menu[/]",
            classes="placeholder-screen",
        )


class Title(Static):
    """A centered title widget."""

    DEFAULT_CSS = """
    Title {
        text-style: bold;
        text-align: center;
        width: 100%;
        padding: 1 0;
    }
    """


class StatusBar(Static):
    """A bar showing daily summary stats."""

    DEFAULT_CSS = """
    StatusBar {
        text-align: center;
        width: 100%;
        padding: 1 0;
        border-top: solid $primary;
    }
    """
