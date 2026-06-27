# cli-tool

A Python CLI tool with Deepseek API integration featuring a fully interactive Textual-based chat TUI.

## Features

- **Chat TUI** — interactive terminal chat with streaming responses, conversation history, syntax-highlighted message bubbles, and auto-scroll
- **CLI chat** — one-shot or streaming completions directly from the command line
- **Config system** — API keys via environment variables or YAML config files
- **Robust API client** — `httpx`-based `DeepseekClient` with retries, connection/timeout error handling, and OpenAI SDK compatibility

## Prerequisites

- **Python** 3.10 or later
- A [Deepseek API key](https://platform.deepseek.com/api_keys)

## Installation

### From source (development install)

```bash
git clone <repo-url>
cd cli-tool
pip install -e .
```

### Optional development dependencies

```bash
pip install -e ".[dev]"
```

This installs `pytest` (tests) and `ruff` (linting).

## Configuration

The tool looks for configuration in two places. Settings from both sources are merged; the environment variable takes precedence for the API key.

### 1. Environment variable

```bash
export DEEPSEEK_API_KEY="sk-your-api-key-here"
```

### 2. YAML config file

Create a YAML file at one of these locations:

- `~/.deepseek.yaml`
- `~/.config/deepseek/config.yaml`

The **first file found** is used.

```yaml
# ~/.deepseek.yaml
api_key: sk-your-api-key-here
model: deepseek-chat        # optional, default: deepseek-chat
temperature: 0.7            # optional, default: 0.7
max_tokens: 4096            # optional, default: 4096
```

All keys are optional. Values not set in the file fall back to the defaults above. The `api_key` from a YAML file is only used if `DEEPSEEK_API_KEY` is not set in the environment.

## Usage

### `cli-tool hello [NAME]`

A simple sanity check.

```bash
$ cli-tool hello
Hello, World!

$ cli-tool hello Alice
Hello, Alice!
```

### `cli-tool chat`

Launches the **interactive Textual TUI**. This is the primary interface.

```bash
$ cli-tool chat
```

See [TUI Features](#tui-features) below for details.

### `cli-tool deepseek chat [OPTIONS] PROMPT`

Send a single prompt to the Deepseek API and stream the response to stdout.

```bash
# Basic prompt (streaming by default)
$ cli-tool deepseek chat "Explain quantum computing in one paragraph"

# With a system message
$ cli-tool deepseek chat -s "You are a helpful assistant" "Hello"

# Custom model and temperature
$ cli-tool deepseek chat -m deepseek-chat -t 0.3 "Write a haiku about trains"

# Non-streaming (returns the full response at once)
$ cli-tool deepseek chat --no-stream "Summarize the TCP handshake"

# Limit output length
$ cli-tool deepseek chat --max-tokens 256 "Name three programming paradigms"
```

#### Options for `deepseek chat`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-m, --model` | string | `deepseek-chat` | Model name |
| `-s, --system` | string | — | System prompt |
| `-t, --temperature` | float | — | Sampling temperature (0–2) |
| `--max-tokens` | int | — | Maximum completion tokens |
| `--no-stream` | flag | off | Disable streaming |

### Running as a module

```bash
python -m cli_tool chat
python -m cli_tool deepseek chat "Hello"
```

## TUI Features

The interactive chat TUI (`cli-tool chat`) is built with [Textual](https://textual.textualize.io/) and provides:

| Feature | Description |
|---------|-------------|
| **Streaming responses** | Assistant messages appear token-by-token as they arrive from the API — no waiting for the full response. |
| **Conversation history** | All messages in the current session are sent as context to the API. Multi-turn conversations work naturally. |
| **Message bubbles** | User messages are shown in green, assistant messages in blue, and errors in red. |
| **Auto-scroll** | The chat view automatically scrolls to the latest message during streaming and after each response. |
| **Threaded I/O** | API calls run in a background thread via Textual's `@work(thread=True)` decorator so the UI never freezes. |
| **Graceful error handling** | Authentication, rate-limit, timeout, and connection errors are caught and displayed inline as red error messages without crashing the TUI. |
| **Configuration-aware** | The TUI reads the same `Config` singleton. If `DEEPSEEK_API_KEY` is not set, it shows a warning banner. |

## Keyboard Shortcuts (TUI)

| Key | Context | Action |
|-----|---------|--------|
| `Ctrl+Q` | Anywhere | Quit the TUI |
| `Enter` | Input field focused | Send message |
| `Tab` | Anywhere | Cycle focus between input and other widgets |

## Project Structure

```
├── README.md
├── pyproject.toml
├── test_tui_scroll.py          # Standalone Textual scroll test
├── src/
│   └── cli_tool/
│       ├── __init__.py         # Package root, re-exports Config
│       ├── __main__.py         # python -m cli_tool entry
│       ├── cli.py              # Click CLI group (main entry point)
│       ├── config.py           # Config dataclass + YAML/env loading
│       ├── tui.py              # Textual ChatApp
│       └── deepseek/
│           ├── __init__.py     # Public API surface
│           ├── client.py       # DeepseekClient (httpx + retries)
│           ├── exceptions.py   # Exception hierarchy
│           └── types.py        # Response dataclasses
└── tests/
    ├── __init__.py
    ├── test_cli.py
    └── test_config.py
```

## Troubleshooting

### `Error: Invalid API key` (401)

- Verify your `DEEPSEEK_API_KEY` environment variable is set and exported correctly.
- If using a YAML config file, check that the key is spelled `api_key` (snake_case) and the value is quoted.
- Try regenerating your key at [platform.deepseek.com/api_keys](https://platform.deepseek.com/api_keys).

### `Error: Rate limit exceeded. Try again later.` (429)

Wait and retry. The Deepseek API enforces rate limits. If this happens frequently, check your plan limits or spread requests further apart.

### `Error: Request timed out.`

This can happen when the model takes too long to generate a response (especially with high `max_tokens` or complex prompts). The client timeout defaults to **600 seconds**. If you need longer, create a `DeepseekClient` with a higher `timeout`:

```python
from cli_tool.deepseek import DeepseekClient

client = DeepseekClient(api_key=..., timeout=1200)
```

### `Error: Could not connect to Deepseek API.`

- Check your internet connection.
- Verify you can reach `https://api.deepseek.com` (e.g., `curl -I https://api.deepseek.com`).
- If behind a proxy, set the `HTTP_PROXY` / `HTTPS_PROXY` environment variables — `httpx` respects them.

### TUI does not start or looks garbled

- The TUI requires a proper terminal with Unicode support. Ensure you are running in a modern terminal (iTerm2, Terminal.app, Windows Terminal, Alacritty, etc.).
- Try running `echo $TERM` — it should report something like `xterm-256color` or `screen-256color`.
- If running inside tmux or screen, ensure Unicode and 256-color support are enabled.

### `ModuleNotFoundError: No module named 'textual'`

Install the package with its dependencies:

```bash
pip install -e .
```

Or install Textual explicitly:

```bash
pip install "textual>=0.40.0"
```

### `DEEPSEEK_API_KEY environment variable is not set`

Either export the variable in your shell profile, or create a YAML config file at `~/.deepseek.yaml` with your key. See the [Configuration](#configuration) section.

### Running tests

```bash
pytest tests/ -v
```

## License

MIT
