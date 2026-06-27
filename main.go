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
			panic(r)
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
