package agent

import (
	"context"
	"fmt"
	"os/exec"
)

type baseRunner struct {
	name string
}

func (b *baseRunner) Name() string { return b.name }

func runCommand(ctx context.Context, name string, args []string, workspace string) (<-chan OutputChunk, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workspace

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	ch := make(chan OutputChunk, 64)

	go func() {
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				ch <- OutputChunk{IsFinal: true, Error: fmt.Errorf("agent panicked: %v", r)}
			}
		}()
		done := make(chan struct{}, 2)

		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := stdout.Read(buf)
				if n > 0 {
					ch <- OutputChunk{Text: string(buf[:n])}
				}
				if err != nil {
					break
				}
			}
			done <- struct{}{}
		}()

		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := stderr.Read(buf)
				if n > 0 {
					ch <- OutputChunk{Text: string(buf[:n])}
				}
				if err != nil {
					break
				}
			}
			done <- struct{}{}
		}()

		<-done
		<-done

		err := cmd.Wait()
		ch <- OutputChunk{IsFinal: true, Error: err}
	}()

	return ch, nil
}
