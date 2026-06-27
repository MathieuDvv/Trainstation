package agent

import (
	"fmt"

	"trainstation/config"
)

type Registry struct {
	agents map[string]Agent
}

func NewRegistry(cfg *config.Config) (*Registry, error) {
	r := &Registry{agents: make(map[string]Agent)}

	if cfg.Agents.Claude.Enabled {
		r.agents["claude"] = NewClaudeRunner(
			cfg.Agents.Claude.SkipPermissions,
			cfg.Agents.Claude.ExtraArgs,
		)
	}
	if cfg.Agents.OpenCode.Enabled {
		r.agents["opencode"] = NewOpenCodeRunner(
			cfg.Agents.OpenCode.SkipPermissions,
			cfg.Agents.OpenCode.ExtraArgs,
		)
	}
	if cfg.Agents.Codex.Enabled {
		r.agents["codex"] = NewCodexRunner(
			cfg.Agents.Codex.SkipPermissions,
			cfg.Agents.Codex.ExtraArgs,
		)
	}
	if cfg.Agents.Antigravity.Enabled {
		r.agents["antigravity"] = NewAntigravityRunner(
			cfg.Agents.Antigravity.SkipPermissions,
			cfg.Agents.Antigravity.ExtraArgs,
		)
	}

	if len(r.agents) == 0 {
		return nil, fmt.Errorf("no agents enabled in configuration")
	}

	return r, nil
}

func (r *Registry) Get(name string) (Agent, error) {
	a, ok := r.agents[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownAgent, name)
	}
	return a, nil
}

func (r *Registry) Available() []string {
	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}

func (r *Registry) Strengths(cfg *config.Config) map[string][]string {
	return map[string][]string{
		"claude":      cfg.Agents.Claude.Strengths,
		"opencode":    cfg.Agents.OpenCode.Strengths,
		"codex":       cfg.Agents.Codex.Strengths,
		"antigravity": cfg.Agents.Antigravity.Strengths,
	}
}
