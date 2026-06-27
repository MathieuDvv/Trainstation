package agent

import (
	"context"
)

type OutputChunk struct {
	Text    string
	IsFinal bool
	Error   error
}

type Agent interface {
	Name() string
	Run(ctx context.Context, workspace string, prompt string) (<-chan OutputChunk, error)
}
