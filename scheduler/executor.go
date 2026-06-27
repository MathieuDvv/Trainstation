package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"trainstation/agent"
	"trainstation/router"
)

type EventType int

const (
	EventPlan EventType = iota
	EventTaskStart
	EventTaskOutput
	EventTaskDone
	EventTaskError
	EventAllDone
)

type Event struct {
	Type      EventType
	TaskID    int
	Agent     string
	Text      string
	Reasoning string
	Tasks     []*Task
	Err       error
}

type Executor struct {
	registry   *agent.Registry
	workflow   *Workflow
	workspace  string
	events     chan Event
	maxParallel int
}

func NewExecutor(registry *agent.Registry, workspace string, maxParallel int) *Executor {
	if maxParallel <= 0 {
		maxParallel = 4
	}
	return &Executor{
		registry:   registry,
		workspace:  workspace,
		events:     make(chan Event, 128),
		maxParallel: maxParallel,
	}
}

func (e *Executor) Events() <-chan Event {
	return e.events
}

func (e *Executor) Execute(ctx context.Context, plan *router.TaskPlan) error {
	wf, err := BuildWorkflow(plan)
	if err != nil {
		return fmt.Errorf("failed to build workflow: %w", err)
	}
	e.workflow = wf

	e.events <- Event{
		Type:      EventPlan,
		Reasoning: plan.Reasoning,
		Tasks:     wf.Tasks(),
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, e.maxParallel)
	completionCh := make(chan int, e.maxParallel)

	for !wf.AllDone() {
		ready := wf.ReadyTasks()

		for _, task := range ready {
			task := task
			wf.MarkRunning(task.ID)

			wg.Add(1)
			sem <- struct{}{}

			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				defer func() { completionCh <- task.ID }()

				e.runTask(ctx, task)
			}()
		}

		if len(ready) == 0 && !wf.AllDone() {
			select {
			case <-completionCh:
			case <-ctx.Done():
				wg.Wait()
				return ctx.Err()
			}
		}
	}

	wg.Wait()

	e.events <- Event{Type: EventAllDone, Tasks: wf.Tasks()}
	close(e.events)
	return nil
}

func (e *Executor) runTask(ctx context.Context, task *Task) {
	defer func() {
		if r := recover(); r != nil {
			e.events <- Event{
				Type:   EventTaskError,
				TaskID: task.ID,
				Agent:  task.Agent,
				Err:    fmt.Errorf("task panicked: %v", r),
			}
			e.workflow.MarkDone(task.ID, "", fmt.Errorf("panic: %v", r))
		}
	}()

	ag, err := e.registry.Get(task.Agent)
	if err != nil {
		e.events <- Event{
			Type:   EventTaskError,
			TaskID: task.ID,
			Agent:  task.Agent,
			Err:    err,
		}
		e.workflow.MarkDone(task.ID, "", err)
		return
	}

	prompt := task.Description
	if task.ParentOutputs != "" {
		prompt = fmt.Sprintf("%s\n\nPrevious task outputs:\n%s", task.Description, task.ParentOutputs)
	}

	e.events <- Event{
		Type:   EventTaskStart,
		TaskID: task.ID,
		Agent:  task.Agent,
		Text:   task.Description,
	}

	ch, err := ag.Run(ctx, e.workspace, prompt)
	if err != nil {
		e.events <- Event{
			Type:   EventTaskError,
			TaskID: task.ID,
			Agent:  task.Agent,
			Err:    err,
		}
		e.workflow.MarkDone(task.ID, "", err)
		return
	}

	var output strings.Builder
	for chunk := range ch {
		if chunk.Error != nil && chunk.IsFinal {
			e.events <- Event{
				Type:   EventTaskError,
				TaskID: task.ID,
				Agent:  task.Agent,
				Err:    chunk.Error,
			}
			e.workflow.MarkDone(task.ID, output.String(), chunk.Error)
			return
		}
		if chunk.Text != "" {
			output.WriteString(chunk.Text)
			e.events <- Event{
				Type:   EventTaskOutput,
				TaskID: task.ID,
				Agent:  task.Agent,
				Text:   chunk.Text,
			}
		}
	}

	e.events <- Event{
		Type:   EventTaskDone,
		TaskID: task.ID,
		Agent:  task.Agent,
		Text:   output.String(),
	}
	e.workflow.MarkDone(task.ID, output.String(), nil)
}
