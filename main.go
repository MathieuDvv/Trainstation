package main

import (
	"fmt"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"

	"trainstation/agent"
	"trainstation/config"
	"trainstation/router"
	"trainstation/tui"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			errBytes := []byte(fmt.Sprintf("Panic caught in main:\n%v\n\n%s\n", r, string(debug.Stack())))
			os.WriteFile("crash.log", errBytes, 0644)
			
			// Force restore terminal state (Alt screen off, Cursor on, Mouse off)
			fmt.Print("\x1b[?1049l\x1b[?25h\x1b[?1002l\x1b[?1003l\x1b[?1006l")
			
			fmt.Printf("\n\033[31;1m💥 Oops! Trainstation crashed unexpectedly.\033[0m\n")
			fmt.Printf("\033[33mError:\033[0m %v\n\n", r)
			fmt.Printf("A detailed crash log has been saved to \033[1mcrash.log\033[0m in the current directory.\n")
			fmt.Printf("Please report this issue if it persists.\n\n")
			os.Exit(1)
		}
	}()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg == nil {
		cfg = config.Default()
	}

	needsOnboarding := len(cfg.ConfiguredProviders()) == 0

	if needsOnboarding {
		runOnboarding(cfg)
	}

	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		fmt.Fprintf(os.Stderr, "No configuration found. Please run again.\n")
		os.Exit(1)
	}

	registry, err := agent.NewRegistry(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing agents: %v\n", err)
		os.Exit(1)
	}

	var rtr *router.Router
	rtr, err = router.New(cfg, registry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	model := tui.New(cfg, rtr, registry)

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runOnboarding(cfg *config.Config) {
	m := tui.NewOnboarding(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
