package scheduler

import (
	"fmt"
	"strings"

	"trainstation/router"
)

type TaskState int

const (
	StatePending TaskState = iota
	StateReady
	StateRunning
	StateDone
	StateError
)

func (s TaskState) String() string {
	switch s {
	case StatePending:
		return "pending"
	case StateReady:
		return "ready"
	case StateRunning:
		return "running"
	case StateDone:
		return "done"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

type Task struct {
	router.TaskSpec
	State    TaskState
	Output   strings.Builder
	Err      error
	ParentOutputs string
}

type Workflow struct {
	tasks map[int]*Task
	order []int
}

func BuildWorkflow(plan *router.TaskPlan) (*Workflow, error) {
	wf := &Workflow{
		tasks: make(map[int]*Task),
	}

	for _, spec := range plan.Tasks {
		wf.tasks[spec.ID] = &Task{
			TaskSpec: spec,
			State:    StatePending,
		}
		wf.order = append(wf.order, spec.ID)
	}

	for id, task := range wf.tasks {
		allDepsExist := true
		for _, dep := range task.DependsOn {
			if _, ok := wf.tasks[dep]; !ok {
				allDepsExist = false
				break
			}
		}
		if !allDepsExist {
			return nil, fmt.Errorf("task %d has missing dependencies", id)
		}
		if hasCycle(wf, id, make(map[int]bool), make(map[int]bool)) {
			return nil, fmt.Errorf("dependency cycle detected involving task %d", id)
		}
	}

	wf.updateStates()

	return wf, nil
}

func hasCycle(wf *Workflow, id int, visited, recStack map[int]bool) bool {
	visited[id] = true
	recStack[id] = true

	for _, dep := range wf.tasks[id].DependsOn {
		if !visited[dep] {
			if hasCycle(wf, dep, visited, recStack) {
				return true
			}
		} else if recStack[dep] {
			return true
		}
	}

	recStack[id] = false
	return false
}

func (wf *Workflow) updateStates() {
	for _, task := range wf.tasks {
		if task.State == StateDone || task.State == StateError || task.State == StateRunning {
			continue
		}
		allDepsDone := true
		for _, dep := range task.DependsOn {
			if wf.tasks[dep].State != StateDone {
				allDepsDone = false
				break
			}
		}
		if allDepsDone {
			task.State = StateReady
		}
	}
}

func (wf *Workflow) ReadyTasks() []*Task {
	var ready []*Task
	for _, id := range wf.order {
		task := wf.tasks[id]
		if task.State == StateReady {
			ready = append(ready, task)
		}
	}
	return ready
}

func (wf *Workflow) AllDone() bool {
	for _, task := range wf.tasks {
		if task.State != StateDone && task.State != StateError {
			return false
		}
	}
	return true
}

func (wf *Workflow) MarkRunning(id int) {
	if t, ok := wf.tasks[id]; ok {
		t.State = StateRunning
	}
}

func (wf *Workflow) MarkDone(id int, output string, err error) {
	if t, ok := wf.tasks[id]; ok {
		if err != nil {
			t.State = StateError
			t.Err = err
		} else {
			t.State = StateDone
		}
		t.Output.WriteString(output)
		wf.updateStates()

		for _, other := range wf.tasks {
			for _, dep := range other.DependsOn {
				if dep == id {
					var sb strings.Builder
					sb.WriteString(other.ParentOutputs)
					sb.WriteString(fmt.Sprintf("--- Output from task %d ---\n%s\n", id, output))
					other.ParentOutputs = sb.String()
				}
			}
		}
	}
}

func (wf *Workflow) Tasks() []*Task {
	tasks := make([]*Task, 0, len(wf.order))
	for _, id := range wf.order {
		tasks = append(tasks, wf.tasks[id])
	}
	return tasks
}

func (wf *Workflow) TaskByID(id int) *Task {
	return wf.tasks[id]
}
