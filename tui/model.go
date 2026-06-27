package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"

	"trainstation/agent"
	"trainstation/config"
	"trainstation/provider"
	"trainstation/router"
	"trainstation/scheduler"
	"trainstation/usage"
)

type appState int

const (
	stateIdle appState = iota
	stateRouting
	stateExecuting
	stateDone
)

type entryKind int

const (
	entryUser entryKind = iota
	entryRouter
	entryAgent
	entryError
	entryInfo
)

type entry struct {
	kind      entryKind
	agent     string
	taskID    int
	taskDesc  string
	text      strings.Builder
	done      bool
	err       error
	startTime time.Time
}

type popupKind int

const (
	popupNone popupKind = iota
	popupHelp
	popupModelPicker
	popupProviderManager
	popupThinkingPicker
	popupAgents
	popupUsage
	popupCommandMenu
)

type popupModel struct {
	kind     popupKind
	selected int
	input    string
	options  []string

	provSection    int    // 0=configured, 1=available
	provSelected   int    // selected index within current section
	addingProvider string // non-empty = in API key input mode
}

type Model struct {
	cfg      *config.Config
	router   *router.Router
	registry *agent.Registry
	ctx      context.Context

	width  int
	height int

	input    textarea.Model
	viewport viewport.Model
	spinner  spinner.Model

	entries []entry

	state        appState
	agentStatus  map[string]string
	usageSnapshot *usage.Snapshot

	executor       *scheduler.Executor
	eventCh        <-chan scheduler.Event
	currentPlan    *router.TaskPlan
	activeTasks    map[int]bool
	completedTasks map[int]bool
	cancelFn       context.CancelFunc
	updateAvailable bool

	popup           popupModel
	showSidebar     bool
	sidebarFocus    bool
	sidebarSelected int
	sidebarMode     string // "agents" or "tasks"

	focusAgent  string
	focusTaskID int // -1 means no task focused

	slashMode    bool
	slashInput   string
	slashMatches []commandDef
}

func New(cfg *config.Config, rtr *router.Router, reg *agent.Registry) Model {
	ta := textarea.New()
	styleTextArea(&ta)
	ta.Placeholder = "Ask anything...  (/ for commands)"
	ta.Focus()
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.SetHeight(1)

	vp := viewport.New(80, 20)
	vp.SetContent("")

	status := make(map[string]string)
	for _, name := range reg.Available() {
		status[name] = "idle"
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(t.warning)

	m := Model{
		cfg:            cfg,
		router:         rtr,
		registry:       reg,
		ctx:            context.Background(),
		input:          ta,
		viewport:       vp,
		spinner:        sp,
		entries:        make([]entry, 0),
		state:          stateIdle,
		agentStatus:    status,
		activeTasks:    make(map[int]bool),
		completedTasks: make(map[int]bool),
		showSidebar:    true,
		focusTaskID:    -1,
		sidebarMode:    "tasks", // default to tasks
	}

	logoStr := strings.Join(logoLines, "\n")
	m.addInfoEntry(logoStr + "\n\nWelcome to Trainstation. Type a task below, or type / for commands.")

	return m
}

func (m Model) checkGitUpdate() tea.Cmd {
	return func() tea.Msg {
		localCmd := exec.Command("git", "rev-parse", "HEAD")
		localOut, err := localCmd.Output()
		if err != nil {
			return gitUpdateMsg{false}
		}

		remoteCmd := exec.Command("git", "ls-remote", "origin", "main")
		remoteOut, err := remoteCmd.Output()
		if err != nil {
			return gitUpdateMsg{false}
		}

		localSha := strings.TrimSpace(string(localOut))
		parts := strings.Fields(string(remoteOut))
		if len(parts) > 0 {
			remoteSha := parts[0]
			return gitUpdateMsg{Available: localSha != remoteSha && remoteSha != ""}
		}
		return gitUpdateMsg{false}
	}
}

type gitUpdateMsg struct {
	Available bool
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.fetchUsage(), m.tick(), m.prefetchModels(), m.checkGitUpdate())
}

type routeResultMsg struct {
	plan *router.TaskPlan
	err  error
}

type eventMsg struct {
	event scheduler.Event
}

type errMsg struct{ err error }

type usageMsg struct {
	snapshot *usage.Snapshot
}

type tickMsg struct{}

type modelsLoadedMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case gitUpdateMsg:
		m.updateAvailable = msg.Available
		if m.updateAvailable {
			m.addInfoEntry(lipgloss.NewStyle().Foreground(t.success).Bold(true).Render("🎉 A new version of Trainstation is available on GitHub!\nType `git pull && go build` or use the system terminal to update."))
			m.refreshViewport()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.showSidebar = m.width > 100
		m.resize()
		return m, nil

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			if m.popup.kind != popupNone {
				return m.handlePopupClick(msg)
			}
			mainWidth := m.width
			if m.showSidebar {
				mainWidth = m.width - sidebarWidth
			}
			if msg.X >= mainWidth {
				// Clicked in sidebar, calculate if an agent was clicked
				yOffset := 7
				if m.cfg.Router.ThinkingLevel != "" {
					yOffset++
				}
				agentOrder := []string{"claude", "codex", "opencode", "antigravity"}
				var activeAgents []string
				for _, name := range agentOrder {
					if _, ok := m.agentStatus[name]; ok {
						activeAgents = append(activeAgents, name)
					}
				}
				if msg.Y >= yOffset && msg.Y < yOffset+len(activeAgents)*4 {
					idx := (msg.Y - yOffset) / 4
					if idx >= 0 && idx < len(activeAgents) {
						m.sidebarMode = "agents"
						m.sidebarSelected = idx
						m.focusAgent = activeAgents[idx]
						m.focusTaskID = -1
						m.resize()
						m.refreshViewport()
					}
				} else if m.currentPlan != nil {
					// Tasks start after Agents and Progress
					// PROGRESS section takes 4 lines (header + 1 blank + progress bar + 1 blank)
					// TASKS section header takes 2 lines (header + 1 blank)
					yOffsetTasks := yOffset + len(activeAgents)*4 + 6
					if msg.Y >= yOffsetTasks {
						idx := (msg.Y - yOffsetTasks) / 2
						if idx >= 0 && idx < len(m.currentPlan.Tasks) {
							m.sidebarMode = "tasks"
							m.sidebarSelected = idx
							m.focusTaskID = m.currentPlan.Tasks[idx].ID
							m.focusAgent = ""
							m.resize()
							m.refreshViewport()
						}
					}
				}
			}
		}
		
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		if m.state == stateRouting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if m.popup.kind != popupNone {
			return m.handlePopupKey(msg)
		}
		return m.handleKey(msg)

	case routeResultMsg:
		if msg.err != nil {
			m.state = stateIdle
			m.addErrorEntry(fmt.Sprintf("Router failed: %v", msg.err))
			m.input.Focus()
			return m, nil
		}
		m.currentPlan = msg.plan
		m.addRouterEntry(msg.plan.Reasoning, msg.plan.Tasks)
		m.state = stateExecuting
		m.input.Blur()
		m.activeTasks = make(map[int]bool)
		m.completedTasks = make(map[int]bool)
		ctx, cancel := context.WithCancel(context.Background())
		m.cancelFn = cancel
		executor := scheduler.NewExecutor(m.registry, m.cfg.Workspace, 4)
		m.executor = executor
		m.eventCh = executor.Events()
		cmds = append(cmds, m.startExecution(ctx, msg.plan), m.waitForEvent())
		return m, tea.Batch(cmds...)

	case eventMsg:
		m.handleEvent(msg.event)
		if m.state == stateExecuting && m.eventCh != nil {
			cmds = append(cmds, m.waitForEvent())
		}
		return m, tea.Batch(cmds...)

	case errMsg:
		m.addErrorEntry(msg.err.Error())
		return m, nil

	case usageMsg:
		m.usageSnapshot = msg.snapshot
		return m, nil

	case modelsLoadedMsg:
		m.refreshViewport()
		return m, nil

	case tickMsg:
		return m, tea.Batch(m.fetchUsage(), m.tick())
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.Type {
	case tea.KeyCtrlC:
		if m.cancelFn != nil {
			m.cancelFn()
		}
		return m, tea.Quit

	case tea.KeyEsc:
		if m.focusAgent != "" || m.focusTaskID >= 0 {
			m.focusAgent = ""
			m.focusTaskID = -1
			m.refreshViewport()
			return m, nil
		}
		if m.slashMode {
			m.slashMode = false
			m.slashInput = ""
			return m, nil
		}
		if m.state == stateExecuting && m.cancelFn != nil {
			m.cancelFn()
			m.addInfoEntry("Tasks cancelled by user.")
			m.state = stateDone
			m.input.Focus()
		}
		return m, nil

	case tea.KeyPgUp:
		m.viewport.PageUp()
		return m, nil

	case tea.KeyPgDown:
		m.viewport.PageDown()
		return m, nil

	case tea.KeyUp:
		if m.slashMode && len(m.slashMatches) > 0 {
			if m.popup.selected > 0 {
				m.popup.selected--
			}
			return m, nil
		}
		if m.sidebarFocus {
			if m.sidebarSelected > 0 {
				m.sidebarSelected--
			}
			return m, nil
		}
		if m.state != stateIdle && m.state != stateDone {
			m.viewport.LineUp(1)
			return m, nil
		}

	case tea.KeyDown:
		if m.slashMode && len(m.slashMatches) > 0 {
			if m.popup.selected < len(m.slashMatches)-1 {
				m.popup.selected++
			}
			return m, nil
		}
		if m.sidebarFocus {
			if m.sidebarSelected < len(m.activeAgents())-1 {
				m.sidebarSelected++
			}
			return m, nil
		}
		if m.state != stateIdle && m.state != stateDone {
			m.viewport.LineDown(1)
			return m, nil
		}

	case tea.KeyRight, tea.KeyLeft:
		if m.sidebarFocus {
			if m.sidebarMode == "agents" {
				m.sidebarMode = "tasks"
			} else {
				m.sidebarMode = "agents"
			}
			m.sidebarSelected = 0
			return m, nil
		}

	case tea.KeyTab:
		if m.showSidebar {
			m.sidebarFocus = !m.sidebarFocus
			if m.sidebarFocus {
				m.input.Blur()
			} else {
				m.input.Focus()
			}
		}
		return m, nil

	case tea.KeyEnter:
		if m.sidebarFocus {
			if m.sidebarMode == "agents" {
				agents := m.activeAgents()
				if m.sidebarSelected >= 0 && m.sidebarSelected < len(agents) {
					m.focusAgent = agents[m.sidebarSelected]
					m.focusTaskID = -1
					m.resize()
					m.refreshViewport()
				}
			} else if m.sidebarMode == "tasks" && m.currentPlan != nil {
				tasks := m.currentPlan.Tasks
				if m.sidebarSelected >= 0 && m.sidebarSelected < len(tasks) {
					m.focusTaskID = tasks[m.sidebarSelected].ID
					m.focusAgent = ""
					m.resize()
					m.refreshViewport()
				}
			}
			return m, nil
		}

		if m.state == stateIdle || m.state == stateDone {
			input := strings.TrimSpace(m.input.Value())
			if input == "" {
				return m, nil
			}

			if m.slashMode && len(m.slashMatches) > 0 {
				idx := m.popup.selected
				if idx >= len(m.slashMatches) || idx < 0 {
					idx = 0
				}
				cmd := m.slashMatches[idx]
				m.input.SetValue("/" + cmd.name + " ")
				m.input.SetCursor(len(m.input.Value()))
				m.slashMode = false
				m.slashInput = ""
				return m, nil
			}

			if strings.HasPrefix(input, "/") {
				m.input.Reset()
				m.slashMode = false
				m.slashInput = ""
				_, err := m.handleSlashCommand(input)
				if err != nil {
					m.addErrorEntry(err.Error())
				}
				return m, nil
			}

			m.input.Reset()
			m.input.Blur()
			m.state = stateRouting
			m.addUserEntry(input)
			return m, tea.Batch(m.routePrompt(input), m.spinner.Tick)
		}

	default:
		if m.state == stateIdle || m.state == stateDone {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)

			currentVal := m.input.Value()
			if strings.HasPrefix(currentVal, "/") {
				m.slashMode = true
				m.slashInput = currentVal
				m.slashMatches = m.getMatchingCommands(currentVal)
				if m.popup.selected >= len(m.slashMatches) {
					if len(m.slashMatches) > 0 {
						m.popup.selected = len(m.slashMatches) - 1
					} else {
						m.popup.selected = 0
					}
				}
			} else {
				m.slashMode = false
			}
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handlePopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		if m.popup.kind == popupProviderManager && m.popup.addingProvider != "" {
			m.popup.addingProvider = ""
			m.popup.input = ""
			return m, nil
		}
		m.popup = popupModel{kind: popupNone}
		m.input.Focus()
		return m, nil

	case tea.KeyTab:
		if m.popup.kind == popupProviderManager && m.popup.addingProvider == "" {
			m.popup.provSection = 1 - m.popup.provSection
			m.popup.provSelected = 0
			return m, nil
		}

	case tea.KeyUp:
		if m.popup.kind == popupProviderManager && m.popup.addingProvider == "" {
			n := m.provSectionItems()
			if m.popup.provSelected > 0 {
				m.popup.provSelected--
			} else if n > 0 {
				m.popup.provSelected = n - 1
			}
			return m, nil
		}
		if len(m.popup.options) > 0 {
			if m.popup.selected > 0 {
				m.popup.selected--
			}
			return m, nil
		}

	case tea.KeyDown:
		if m.popup.kind == popupProviderManager && m.popup.addingProvider == "" {
			n := m.provSectionItems()
			if m.popup.provSelected < n-1 {
				m.popup.provSelected++
			} else if n > 0 {
				m.popup.provSelected = 0
			}
			return m, nil
		}
		if len(m.popup.options) > 0 {
			if m.popup.selected < len(m.popup.options)-1 {
				m.popup.selected++
			}
			return m, nil
		}

	case tea.KeyEnter:
		if m.popup.kind == popupProviderManager {
			if m.popup.addingProvider != "" {
				key := strings.TrimSpace(m.popup.input)
				if key == "" {
					def := provider.Get(m.popup.addingProvider)
					if def != nil {
						if envVal := os.Getenv(def.EnvVar); envVal != "" {
							key = envVal
						}
					}
				}
				if key != "" {
					m.cfg.SetProvider(m.popup.addingProvider, key)
					config.Save(m.cfg)
					m.addInfoEntry("Provider added: " + m.popup.addingProvider)
					provider.InvalidateCache(m.popup.addingProvider)
				}
				m.popup.addingProvider = ""
				m.popup.input = ""
				return m, nil
			}
			if m.popup.provSection == 0 {
				return m.removeSelectedProvider()
			}
			return m.addSelectedProvider()
		}
		if m.popup.kind == popupModelPicker || m.popup.kind == popupThinkingPicker {
			input := strings.TrimSpace(m.popup.input)
			if input == "" && len(m.popup.options) > 0 && m.popup.selected >= 0 && m.popup.selected < len(m.popup.options) {
				input = m.popup.options[m.popup.selected]
			}
			if input != "" {
				m.handlePopupInput(input)
			}
			m.popup = popupModel{kind: popupNone}
			m.input.Focus()
			return m, nil
		}
		m.popup = popupModel{kind: popupNone}
		m.input.Focus()
		return m, nil

	case tea.KeyBackspace:
		if m.popup.kind == popupProviderManager && m.popup.addingProvider != "" {
			if len(m.popup.input) > 0 {
				m.popup.input = m.popup.input[:len(m.popup.input)-1]
			}
			return m, nil
		}
		if m.popup.kind == popupModelPicker || m.popup.kind == popupThinkingPicker {
			if len(m.popup.input) > 0 {
				m.popup.input = m.popup.input[:len(m.popup.input)-1]
			}
			return m, nil
		}
	}

	if m.popup.kind == popupProviderManager && m.popup.addingProvider != "" {
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
			m.popup.input += string(msg.Runes)
			return m, nil
		}
		return m, nil
	}
	if m.popup.kind == popupModelPicker || m.popup.kind == popupThinkingPicker {
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
			m.popup.input += string(msg.Runes)
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handlePopupInput(input string) {
	switch m.popup.kind {
	case popupModelPicker:
		parts := strings.SplitN(input, ":", 2)
		if len(parts) == 2 {
			provName := strings.TrimSpace(parts[0])
			modelID := strings.TrimSpace(parts[1])
			def := provider.Get(provName)
			if def == nil {
				m.addErrorEntry("Unknown provider: " + provName)
				return
			}
			// Check dynamic models (from API cache or fallback to hardcoded)
			apiKey := m.cfg.GetAPIKey(provName)
			models := provider.GetModels(m.ctx, provName, apiKey)
			valid := false
			for _, model := range models {
				if model.ID == modelID {
					valid = true
					break
				}
			}
			// Also check hardcoded as fallback
			if !valid {
				for _, model := range def.Models {
					if model.ID == modelID {
						valid = true
						break
					}
				}
			}
			if !valid {
				m.addErrorEntry("Unknown model: " + modelID)
				return
			}
			m.cfg.Router.Provider = provName
			m.cfg.Router.Model = modelID
			config.Save(m.cfg)
			m.addInfoEntry(fmt.Sprintf("Router model: %s / %s", def.Label, provider.ModelLabel(provName, modelID)))
		}

	case popupProviderManager:
		parts := strings.Fields(input)
		if len(parts) >= 2 && parts[0] == "add" {
			provName := parts[1]
			def := provider.Get(provName)
			if def == nil {
				m.addErrorEntry("Unknown provider: " + provName)
				return
			}
			apiKey := ""
			if len(parts) >= 3 {
				apiKey = parts[2]
			} else if envVal := os.Getenv(def.EnvVar); envVal != "" {
				apiKey = envVal
			}
			if apiKey == "" {
				m.addErrorEntry("No API key provided for " + provName)
				return
			}
			m.cfg.SetProvider(provName, apiKey)
			config.Save(m.cfg)
			m.addInfoEntry("Provider added: " + def.Label)
		} else if len(parts) >= 2 && parts[0] == "remove" {
			provName := parts[1]
			delete(m.cfg.Providers, provName)
			config.Save(m.cfg)
			m.addInfoEntry("Provider removed: " + provName)
		}

	case popupThinkingPicker:
		err := m.setThinkingLevel(input)
		if err != nil {
			m.addErrorEntry(err.Error())
		}
	}
}

func (m *Model) handleEvent(event scheduler.Event) {
	switch event.Type {
	case scheduler.EventPlan:
		for _, task := range event.Tasks {
			m.agentStatus[task.Agent] = "queued"
		}

	case scheduler.EventTaskStart:
		m.activeTasks[event.TaskID] = true
		m.agentStatus[event.Agent] = "running"
		m.addAgentEntry(event.TaskID, event.Agent, event.Text)

	case scheduler.EventTaskOutput:
		m.appendAgentOutput(event.TaskID, event.Text)

	case scheduler.EventTaskDone:
		delete(m.activeTasks, event.TaskID)
		m.completedTasks[event.TaskID] = true
		m.agentStatus[event.Agent] = "idle"
		m.markAgentEntryDone(event.TaskID, nil)

	case scheduler.EventTaskError:
		delete(m.activeTasks, event.TaskID)
		m.completedTasks[event.TaskID] = true
		m.agentStatus[event.Agent] = "error"
		m.markAgentEntryDone(event.TaskID, event.Err)

	case scheduler.EventAllDone:
		m.state = stateDone
		for name := range m.agentStatus {
			m.agentStatus[name] = "idle"
		}
		m.addInfoEntry("All tasks completed.")
		m.input.Focus()
	}
	m.refreshViewport()
}

func (m Model) routePrompt(prompt string) tea.Cmd {
	rtr := m.router
	if rtr == nil {
		return func() tea.Msg {
			return routeResultMsg{err: fmt.Errorf("no router configured — use /provider to add an API key")}
		}
	}
	
	if m.usageSnapshot != nil {
		avail := m.usageSnapshot.AvailableAgents()
		if len(avail) > 0 {
			rtr.UpdateAvailable(avail)
		}
	}

	return func() tea.Msg {
		plan, err := rtr.Route(context.Background(), prompt)
		return routeResultMsg{plan: plan, err: err}
	}
}

func (m Model) startExecution(ctx context.Context, plan *router.TaskPlan) tea.Cmd {
	exec := m.executor
	return func() tea.Msg {
		err := exec.Execute(ctx, plan)
		if err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}

func (m Model) waitForEvent() tea.Cmd {
	ch := m.eventCh
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return nil
		}
		return eventMsg{event: event}
	}
}

func (m Model) fetchUsage() tea.Cmd {
	enabled := make(map[string]bool)
	for _, name := range m.registry.Available() {
		enabled[name] = true
	}
	cfg := m.cfg
	return func() tea.Msg {
		snap := usage.FetchAll(context.Background(), cfg, enabled)
		return usageMsg{snapshot: snap}
	}
}

func (m Model) prefetchModels() tea.Cmd {
	cfg := m.cfg
	return func() tea.Msg {
		providers := make(map[string]string)
		for _, name := range cfg.ConfiguredProviders() {
			providers[name] = cfg.GetAPIKey(name)
		}
		provider.PrefetchModels(context.Background(), providers)
		return modelsLoadedMsg{}
	}
}

func (m Model) tick() tea.Cmd {
	return tea.Tick(60*time.Second, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m *Model) resize() {
	if m.width == 0 || m.height == 0 {
		return
	}

	mainWidth := m.width
	if m.showSidebar {
		mainWidth = m.width - sidebarWidth
	}

	m.input.SetWidth(mainWidth - 4)

	vpHeight := m.height - 5
	if vpHeight < 3 {
		vpHeight = 3
	}
	m.viewport.Width = mainWidth - 2
	m.viewport.Height = vpHeight
}

func (m *Model) addUserEntry(text string) {
	e := entry{kind: entryUser, agent: "user", startTime: time.Now()}
	e.text.WriteString(text)
	m.entries = append(m.entries, e)
	m.refreshViewport()
	m.viewport.GotoBottom()
}

func (m *Model) addRouterEntry(reasoning string, tasks []router.TaskSpec) {
	var sb strings.Builder
	sb.WriteString(reasoning + "\n")
	for _, task := range tasks {
		name := agentLabel(task.Agent)
		deps := ""
		if len(task.DependsOn) > 0 {
			depsStrs := make([]string, len(task.DependsOn))
			for i, d := range task.DependsOn {
				depsStrs[i] = fmt.Sprintf("#%d", d)
			}
			deps = fmt.Sprintf(" → after %s", strings.Join(depsStrs, ", "))
		}
		sb.WriteString(fmt.Sprintf("  #%d %s%s\n    %s\n", task.ID, name, deps, task.Description))
	}
	e := entry{kind: entryRouter, agent: "router", startTime: time.Now()}
	e.text.WriteString(sb.String())
	m.entries = append(m.entries, e)
	m.refreshViewport()
}

func (m *Model) addAgentEntry(taskID int, agentName, desc string) {
	e := entry{
		kind:      entryAgent,
		agent:     agentName,
		taskID:    taskID,
		taskDesc:  desc,
		startTime: time.Now(),
	}
	m.entries = append(m.entries, e)
	m.refreshViewport()
}

func (m *Model) appendAgentOutput(taskID int, text string) {
	for i := len(m.entries) - 1; i >= 0; i-- {
		if m.entries[i].kind == entryAgent && m.entries[i].taskID == taskID {
			m.entries[i].text.WriteString(text)
			return
		}
	}
}

func (m *Model) markAgentEntryDone(taskID int, err error) {
	for i := len(m.entries) - 1; i >= 0; i-- {
		if m.entries[i].kind == entryAgent && m.entries[i].taskID == taskID {
			m.entries[i].done = true
			m.entries[i].err = err
			return
		}
	}
}

func (m *Model) addErrorEntry(text string) {
	e := entry{kind: entryError, agent: "error", startTime: time.Now()}
	e.text.WriteString(text)
	m.entries = append(m.entries, e)
	m.refreshViewport()
}

func (m *Model) addInfoEntry(text string) {
	e := entry{kind: entryInfo, agent: "system", startTime: time.Now()}
	e.text.WriteString(text)
	m.entries = append(m.entries, e)
	m.refreshViewport()
}

func (m *Model) refreshViewport() {
	wasAtBottom := m.viewport.AtBottom()
	content := m.renderEntries()
	m.viewport.SetContent(content)
	if wasAtBottom {
		m.viewport.GotoBottom()
	}
}

func (m Model) renderEntries() string {
	var sb strings.Builder
	first := true
	
	wrapWidth := m.viewport.Width - 4
	if wrapWidth <= 0 {
		wrapWidth = m.width - 4
		if m.showSidebar {
			wrapWidth -= sidebarWidth
		}
	}
	if wrapWidth < 10 {
		wrapWidth = 10
	}

	for _, e := range m.entries {
		if m.focusTaskID >= 0 {
			if e.kind == entryAgent && e.taskID != m.focusTaskID {
				continue
			}
		} else if m.focusAgent != "" {
			if e.agent != m.focusAgent && e.agent != "system" && e.agent != "router" {
				continue
			}
		}

		if !first {
			sb.WriteString("\n\n")
		}
		first = false
		switch e.kind {
		case entryUser:
			color := t.text
			bar := leftBar(color)
			label := boldStyle.Foreground(color).Render("You")
			
			wrapped := wordwrap.String(e.text.String(), wrapWidth)
			var bodyStr string
			for i, line := range strings.Split(wrapped, "\n") {
				if i > 0 { bodyStr += "\n" }
				bodyStr += bar + textStyle.Render(line)
			}
			sb.WriteString(bar + label + "\n" + bodyStr)

		case entryRouter:
			color := agentColor("router")
			bar := leftBar(color)
			label := boldStyle.Foreground(color).Render("Router")
			
			wrapped := wordwrap.String(e.text.String(), wrapWidth)
			var bodyStr string
			for i, line := range strings.Split(wrapped, "\n") {
				if i > 0 { bodyStr += "\n" }
				bodyStr += bar + mutedStyle.Render(line)
			}
			sb.WriteString(bar + label + "\n" + bodyStr)

		case entryAgent:
			color := agentColor(e.agent)
			bar := leftBar(color)
			name := agentLabel(e.agent)
			status := ""
			if e.done {
				if e.err != nil {
					status = errorStyle.Render(" ✗")
				} else {
					status = successStyle.Render(" ✓")
				}
			}
			label := boldStyle.Foreground(color).Render(name + status)
			meta := dimStyle.Render(fmt.Sprintf("task #%d", e.taskID))
			desc := ""
			if e.taskDesc != "" {
				desc = mutedStyle.Render(e.taskDesc) + "\n"
			}
			output := strings.TrimRight(e.text.String(), "\n")
			headerLine := bar + label + " " + meta
			descLine := ""
			if desc != "" {
				descLine = bar + desc
			}
			outputLine := ""
			if output != "" {
				wrapped := wordwrap.String(output, wrapWidth)
				for _, line := range strings.Split(wrapped, "\n") {
					outputLine += bar + line + "\n"
				}
				outputLine = strings.TrimRight(outputLine, "\n")
			}
			body := headerLine + "\n" + descLine + outputLine
			sb.WriteString(body)

		case entryError:
			bar := leftBar(t.error)
			wrapped := wordwrap.String("✗ " + e.text.String(), wrapWidth)
			var bodyStr string
			for i, line := range strings.Split(wrapped, "\n") {
				if i > 0 { bodyStr += "\n" }
				bodyStr += bar + errorStyle.Render(line)
			}
			sb.WriteString(bodyStr)

		case entryInfo:
			bar := leftBar(t.textDim)
			wrapped := wordwrap.String(e.text.String(), wrapWidth)
			var bodyStr string
			for i, line := range strings.Split(wrapped, "\n") {
				if i > 0 { bodyStr += "\n" }
				bodyStr += bar + line
			}
			sb.WriteString(bodyStr)
		}
	}
	return sb.String()
}

func getAgentConfig(cfg *config.Config, name string) *config.AgentConfig {
	switch name {
	case "claude":
		return &cfg.Agents.Claude
	case "codex":
		return &cfg.Agents.Codex
	case "opencode":
		return &cfg.Agents.OpenCode
	case "antigravity":
		return &cfg.Agents.Antigravity
	default:
		return nil
	}
}

func (m Model) activeAgents() []string {
	agentOrder := []string{"claude", "codex", "opencode", "antigravity"}
	var active []string
	for _, name := range agentOrder {
		if _, ok := m.agentStatus[name]; ok {
			active = append(active, name)
		}
	}
	return active
}

func getEnv(key string) string {
	return os.Getenv(key)
}

func (m *Model) handlePopupClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	popupW, popupH := m.popupDim()
	popupX := (m.width - popupW) / 2
	popupY := (m.height - popupH) / 2

	if msg.X < popupX || msg.X >= popupX+popupW || msg.Y < popupY || msg.Y >= popupY+popupH {
		m.popup = popupModel{kind: popupNone}
		m.input.Focus()
		return m, nil
	}

	bodyY := msg.Y - popupY - 3
	if bodyY < 0 {
		return m, nil
	}

	switch m.popup.kind {
	case popupModelPicker:
		return m.clickModelPicker(bodyY)
	case popupThinkingPicker:
		return m.clickThinkingPicker(bodyY)
	case popupProviderManager:
		return m.clickProviderManager(bodyY)
	}

	return m, nil
}

func (m *Model) popupDim() (int, int) {
	switch m.popup.kind {
	case popupHelp:
		return 58, 14
	case popupModelPicker:
		return 60, 24
	case popupProviderManager:
		return 62, 24
	case popupThinkingPicker:
		return 44, 20
	case popupAgents:
		return 60, 20
	case popupUsage:
		return 60, 18
	case popupCommandMenu:
		return 60, 14
	default:
		return 50, 10
	}
}

func (m *Model) clickModelPicker(bodyY int) (tea.Model, tea.Cmd) {
	line := 0
	idx := 0
	for _, provName := range m.cfg.ConfiguredProviders() {
		if line == bodyY {
			return m, nil
		}
		line++
		def := provider.Get(provName)
		if def == nil {
			continue
		}
		apiKey := m.cfg.GetAPIKey(provName)
		models := provider.GetModels(m.ctx, provName, apiKey)
		for _, model := range models {
			if line == bodyY {
				m.handlePopupInput(provName + ":" + model.ID)
				m.popup = popupModel{kind: popupNone}
				m.input.Focus()
				return m, nil
			}
			line++
			idx++
		}
		line++
	}
	return m, nil
}

func (m *Model) clickThinkingPicker(bodyY int) (tea.Model, tea.Cmd) {
	idx := bodyY / 3
	if idx >= 0 && idx < 4 {
		levels := []string{"low", "medium", "high", "max"}
		m.handlePopupInput(levels[idx])
		m.popup = popupModel{kind: popupNone}
		m.input.Focus()
	}
	return m, nil
}

func (m *Model) clickProviderManager(bodyY int) (tea.Model, tea.Cmd) {
	if m.popup.addingProvider != "" {
		return m, nil
	}

	if bodyY == 0 {
		m.popup.provSection = 1 - m.popup.provSection
		m.popup.provSelected = 0
		return m, nil
	}

	itemStartY := 2
	if m.popup.provSection == 0 {
		n := len(m.cfg.ConfiguredProviders())
		if bodyY >= itemStartY && bodyY < itemStartY+n {
			m.popup.provSelected = bodyY - itemStartY
			return m.removeSelectedProvider()
		}
	} else {
		avail := m.availableProviders()
		if bodyY >= itemStartY && bodyY < itemStartY+len(avail) {
			m.popup.provSelected = bodyY - itemStartY
			return m.addSelectedProvider()
		}
	}

	return m, nil
}

func (m *Model) availableProviders() []string {
	var avail []string
	for _, def := range provider.Definitions {
		if m.cfg.GetAPIKey(def.Name) == "" {
			avail = append(avail, def.Name)
		}
	}
	return avail
}

func (m *Model) provSectionItems() int {
	if m.popup.provSection == 0 {
		return len(m.cfg.ConfiguredProviders())
	}
	return len(m.availableProviders())
}

func (m *Model) removeSelectedProvider() (tea.Model, tea.Cmd) {
	configured := m.cfg.ConfiguredProviders()
	if m.popup.provSelected < 0 || m.popup.provSelected >= len(configured) {
		return m, nil
	}
	provName := configured[m.popup.provSelected]
	delete(m.cfg.Providers, provName)
	config.Save(m.cfg)
	m.addInfoEntry("Provider removed: " + provName)
	provider.InvalidateCache(provName)

	if m.cfg.Router.Provider == provName {
		remaining := m.cfg.ConfiguredProviders()
		if len(remaining) > 0 {
			m.cfg.Router.Provider = remaining[0]
			newDef := provider.Get(remaining[0])
			if newDef != nil && len(newDef.Models) > 0 {
				m.cfg.Router.Model = newDef.Models[0].ID
			}
			if m.router != nil {
				apiKey := m.cfg.GetAPIKey(remaining[0])
				baseURL := m.cfg.GetBaseURL(remaining[0])
				m.router.UpdateConfig(remaining[0], m.cfg.Router.Model, apiKey, baseURL)
			}
		} else {
			m.cfg.Router.Provider = ""
			m.cfg.Router.Model = ""
		}
	}

	n := len(m.cfg.ConfiguredProviders())
	if n == 0 {
		m.popup.provSection = 1
		m.popup.provSelected = 0
	} else if m.popup.provSelected >= n {
		m.popup.provSelected = max(0, n-1)
	}

	return m, nil
}

func (m *Model) addSelectedProvider() (tea.Model, tea.Cmd) {
	available := m.availableProviders()
	if m.popup.provSelected < 0 || m.popup.provSelected >= len(available) {
		return m, nil
	}
	provName := available[m.popup.provSelected]
	def := provider.Get(provName)
	if def == nil {
		return m, nil
	}
	if envVal := os.Getenv(def.EnvVar); envVal != "" {
		m.cfg.SetProvider(provName, envVal)
		config.Save(m.cfg)
		m.addInfoEntry("Provider added: " + provName + " (from env)")
		provider.InvalidateCache(provName)
		if m.cfg.Router.Provider == "" {
			m.cfg.Router.Provider = provName
			if len(def.Models) > 0 {
				m.cfg.Router.Model = def.Models[0].ID
			}
		}
		if m.popup.provSelected >= len(m.availableProviders()) {
			m.popup.provSelected = max(0, len(m.availableProviders())-1)
		}
	} else {
		m.popup.addingProvider = provName
		m.popup.input = ""
	}
	return m, nil
}
