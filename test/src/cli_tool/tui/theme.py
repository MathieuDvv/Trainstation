CSS = """
Screen {
    align: center middle;
}

#main-menu {
    width: 48;
    height: auto;
    padding: 1 2;
    border: thick $accent;
    background: $surface;
}

#main-title {
    text-style: bold;
    color: $accent;
    content-align: center middle;
    padding: 1 0;
    width: 100%;
}

#main-subtitle {
    content-align: center middle;
    padding-bottom: 1;
    width: 100%;
    color: $text-muted;
}

#main-summary {
    content-align: center middle;
    padding: 1 0;
    width: 100%;
    color: $success;
    border-top: solid $primary;
}

#menu-buttons {
    width: 100%;
    padding: 1 0;
}

.menu-btn {
    width: 100%;
    margin-bottom: 1;
}

.placeholder-screen {
    width: 60;
    height: auto;
    padding: 2 3;
    border: thick $accent;
    background: $surface;
}

.placeholder-title {
    text-style: bold;
    color: $secondary;
    content-align: center middle;
    padding-bottom: 1;
    width: 100%;
}

.placeholder-body {
    padding: 1 0;
    width: 100%;
    color: $text-muted;
}

.placeholder-footer {
    padding: 1 0;
    color: $success;
    content-align: center middle;
    width: 100%;
}

#help-overlay {
    width: 56;
    height: auto;
    padding: 1 2;
    border: thick $accent;
    background: $surface;
}

#help-title {
    text-style: bold;
    color: $warning;
    content-align: center middle;
    padding-bottom: 1;
    width: 100%;
}

.help-section {
    padding: 1 0;
    width: 100%;
    color: $text;
}

Dialog {
    align: center middle;
}

#dialog-container {
    width: 50;
    height: auto;
    padding: 1 2;
    border: thick $warning;
    background: $surface;
}

#dialog-title {
    text-style: bold;
    color: $warning;
    content-align: center middle;
    padding-bottom: 1;
    width: 100%;
}

#dialog-message {
    padding: 1 0;
    width: 100%;
    color: $text;
    content-align: center middle;
}

#dialog-buttons {
    width: 100%;
    padding-top: 1;
    align: center middle;
}

.dialog-btn {
    margin: 0 1;
}
"""
