# 🚂 Trainstation

```
     ████████╗██████╗  █████╗ ██╗███╗   ██╗███████╗████████╗ █████╗ ████████╗██╗ ██████╗ ███╗   ██╗
     ╚══██╔══╝██╔══██╗██╔══██╗██║████╗  ██║██╔════╝╚══██╔══╝██╔══██╗╚══██╔══╝██║██╔═══██╗████╗  ██║
        ██║   ██████╔╝███████║██║██╔██╗ ██║███████╗   ██║   ███████║   ██║   ██║██║   ██║██╔██╗ ██║
        ██║   ██╔══██╗██╔══██║██║██║╚██╗██║╚════██║   ██║   ██╔══██║   ██║   ██║██║   ██║██║╚██╗██║
        ██║   ██║  ██║██║  ██║██║██║ ╚████║███████║   ██║   ██║  ██║   ██║   ██║╚██████╔╝██║ ╚████║
        ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝   ╚═╝  ╚═╝   ╚═╝   ╚═╝ ╚═════╝ ╚═╝  ╚═══╝

                              ┌──────────────────────────────────┐
                              │    AI Agent Orchestration Hub    │
                              └──────────────────────────────────┘
```

Trainstation is an AI agent orchestrator that breaks down complex tasks and dispatches them to specialized coding agents — [Claude Code](https://docs.anthropic.com/en/docs/claude-code), [OpenCode](https://github.com/anomalyco/opencode), [Codex](https://openai.com/index/introducing-codex-cli/), and [Antigravity](https://antigravity.dev) — running in parallel.

## How it works

```
   You  ──→  Router (LLM)  ──→  Task Plan  ──→  Scheduler  ──→  Agents (parallel)
                                                 │
                                          ┌──────┼──────┐
                                          ▼      ▼      ▼
                                       Claude  Codex  OpenCode
```

1. **You** describe what you want in natural language
2. The **Router** (powered by an LLM) analyzes the request and produces a task plan with dependencies
3. The **Scheduler** executes tasks in parallel across agents, respecting dependency order
4. Each **Agent** receives its task with context from upstream tasks and streams output in real time

## Features

- **Multi-agent parallelism** — run Claude Code, Codex, OpenCode, and Antigravity simultaneously
- **Smart routing** — an LLM decomposes your request into specialized subtasks with dependencies
- **Live TUI** — terminal interface with real-time streaming, agent sidebar, and task progress
- **Context passing** — outputs from upstream tasks are fed to dependent downstream tasks
- **Provider agnostic** — use OpenAI, Anthropic, Google, or any OpenAI-compatible API for the router

## Commands

| Command            | Description                          |
|--------------------|--------------------------------------|
| `/help`            | Show all commands                    |
| `/model`           | Change the router model              |
| `/provider`        | Add, remove, or change API providers |
| `/thinking`        | Set reasoning level (low/medium/high)|
| `/agents`          | Enable or disable agents             |
| `/strengths`       | Edit agent strengths (controls routing) |
| `/workspace`       | Set the working directory            |
| `/usage`           | View agent API usage & rate limits   |
| `Esc`              | Cancel running tasks                 |
| `Tab`              | Toggle sidebar focus                 |

## Quick start

```bash
# Clone
git clone https://github.com/MathieuDvv/Trainstation.git
cd Trainstation

# Build
go build -o trainstation .

# Run — onboarding will guide you through setup on first launch
./trainstation
```

On first run, Trainstation will walk you through adding at least one AI provider API key. You can configure which coding agents to enable, set their strengths for routing, and choose the router model.

## Configuration

Configuration is stored in `~/.config/trainstation/config.json`. See [config/config.go](config/config.go) for all options.

## License

MIT
