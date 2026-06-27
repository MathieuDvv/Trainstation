import click


@click.group()
@click.version_option()
def main():
    """Trainstation — a language learning TUI application."""


@main.command()
def study():
    """Launch the interactive study TUI."""
    from cli_tool.tui import TrainApp

    app = TrainApp()
    app.run()


@main.command()
def stats():
    """Show your study statistics."""
    click.echo("Stats: not yet implemented.")


@main.command()
def import_data():
    """Import vocabulary from a CSV or JSON file."""
    click.echo("Import: not yet implemented.")


@main.command()
def export():
    """Export vocabulary and stats."""
    click.echo("Export: not yet implemented.")


if __name__ == "__main__":
    main()
