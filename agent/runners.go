package agent

import (
	"context"
	"fmt"
)

type ClaudeRunner struct {
	baseRunner
	SkipPermissions bool
	ExtraArgs       []string
}

func NewClaudeRunner(skip bool, extra []string) *ClaudeRunner {
	return &ClaudeRunner{
		baseRunner:      baseRunner{name: "claude"},
		SkipPermissions: skip,
		ExtraArgs:       extra,
	}
}

func (c *ClaudeRunner) Run(ctx context.Context, workspace string, prompt string) (<-chan OutputChunk, error) {
	args := []string{"--print"}
	if c.SkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}
	args = append(args, c.ExtraArgs...)
	args = append(args, prompt)
	return runCommand(ctx, "claude", args, workspace)
}

type OpenCodeRunner struct {
	baseRunner
	SkipPermissions bool
	ExtraArgs       []string
}

func NewOpenCodeRunner(skip bool, extra []string) *OpenCodeRunner {
	return &OpenCodeRunner{
		baseRunner:      baseRunner{name: "opencode"},
		SkipPermissions: skip,
		ExtraArgs:       extra,
	}
}

func (o *OpenCodeRunner) Run(ctx context.Context, workspace string, prompt string) (<-chan OutputChunk, error) {
	args := []string{"run"}
	args = append(args, o.ExtraArgs...)
	args = append(args, prompt)
	return runCommand(ctx, "opencode", args, workspace)
}

type CodexRunner struct {
	baseRunner
	SkipPermissions bool
	ExtraArgs       []string
}

func NewCodexRunner(skip bool, extra []string) *CodexRunner {
	return &CodexRunner{
		baseRunner:      baseRunner{name: "codex"},
		SkipPermissions: skip,
		ExtraArgs:       extra,
	}
}

func (c *CodexRunner) Run(ctx context.Context, workspace string, prompt string) (<-chan OutputChunk, error) {
	args := []string{"exec"}
	if c.SkipPermissions {
		args = append(args, "--full-auto")
	}
	args = append(args, c.ExtraArgs...)
	args = append(args, prompt)
	return runCommand(ctx, "codex", args, workspace)
}

type AntigravityRunner struct {
	baseRunner
	SkipPermissions bool
	ExtraArgs       []string
}

func NewAntigravityRunner(skip bool, extra []string) *AntigravityRunner {
	return &AntigravityRunner{
		baseRunner:      baseRunner{name: "antigravity"},
		SkipPermissions: skip,
		ExtraArgs:       extra,
	}
}

func (a *AntigravityRunner) Run(ctx context.Context, workspace string, prompt string) (<-chan OutputChunk, error) {
	args := []string{"--print", prompt}
	if a.SkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}
	if workspace != "" {
		args = append(args, "--add-dir", workspace)
	}
	args = append(args, a.ExtraArgs...)
	return runCommand(ctx, "agy", args, workspace)
}

var ErrUnknownAgent = fmt.Errorf("unknown agent")
