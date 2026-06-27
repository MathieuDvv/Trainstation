import os
import sys

import click

from .deepseek import (
    DeepseekAPIError,
    DeepseekAuthError,
    DeepseekClient,
    DeepseekConnectionError,
    DeepseekRateLimitError,
    DeepseekTimeoutError,
)


@click.group()
@click.version_option()
def main():
    """A Python CLI tool with Deepseek API integration."""


@main.command()
@click.argument("name", default="World")
def hello(name):
    """Say hello."""
    click.echo(f"Hello, {name}!")


@main.command()
def chat():
    """Start the interactive chat TUI."""
    from .tui import ChatApp

    app = ChatApp()
    app.run()


# ---------------------------------------------------------------------------
# Deepseek chat completions commands
# ---------------------------------------------------------------------------


@main.group(name="deepseek")
def deepseek_group():
    """Interact with the Deepseek API."""


def _get_client() -> DeepseekClient:
    api_key = os.environ.get("DEEPSEEK_API_KEY")
    if not api_key:
        raise click.UsageError("DEEPSEEK_API_KEY environment variable is not set.")
    return DeepseekClient(api_key=api_key)


@deepseek_group.command("chat")
@click.option(
    "-m", "--model", default="deepseek-chat", show_default=True, help="Model name"
)
@click.option("-s", "--system", default=None, help="System prompt")
@click.option(
    "-t",
    "--temperature",
    type=float,
    default=None,
    help="Sampling temperature",
)
@click.option("--max-tokens", type=int, default=None, help="Max completion tokens")
@click.option("--no-stream", is_flag=True, help="Disable streaming mode")
@click.argument("prompt")
def deepseek_chat(model, system, temperature, max_tokens, no_stream, prompt):
    """Send a chat completion request to Deepseek."""
    messages = []
    if system:
        messages.append({"role": "system", "content": system})
    messages.append({"role": "user", "content": prompt})

    try:
        client = _get_client()
        if no_stream:
            completion = client.chat_completion(
                model=model,
                messages=messages,
                temperature=temperature,
                max_tokens=max_tokens,
            )
            click.echo(completion.choices[0].message.content)
        else:
            for chunk in client.chat_completion_stream(
                model=model,
                messages=messages,
                temperature=temperature,
                max_tokens=max_tokens,
            ):
                content = chunk.choices[0].message.content
                if content:
                    sys.stdout.write(content)
                    sys.stdout.flush()
            sys.stdout.write("\n")
    except DeepseekAuthError:
        click.echo("Error: Invalid API key.", err=True)
        raise SystemExit(1)
    except DeepseekRateLimitError:
        click.echo("Error: Rate limit exceeded. Try again later.", err=True)
        raise SystemExit(1)
    except DeepseekTimeoutError:
        click.echo("Error: Request timed out.", err=True)
        raise SystemExit(1)
    except DeepseekConnectionError:
        click.echo("Error: Could not connect to Deepseek API.", err=True)
        raise SystemExit(1)
    except DeepseekAPIError as exc:
        click.echo(f"Error ({exc.status_code}): {exc}", err=True)
        raise SystemExit(1)


if __name__ == "__main__":
    main()
