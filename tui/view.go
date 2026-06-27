package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"

	"trainstation/provider"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	mainWidth := m.width
	if m.showSidebar {
		mainWidth = m.width - sidebarWidth
	}

	main := m.renderMain(mainWidth)

	var screen string
	if m.showSidebar {
		sidebar := m.renderSidebar()
		screen = lipgloss.JoinHorizontal(lipgloss.Top, main, sidebar)
	} else {
		screen = main
	}

	screen = lipgloss.NewStyle().
		Background(t.bg).
		Width(m.width).
		Height(m.height).
		Render(screen)

	if m.popup.kind != popupNone {
		popupStr := m.renderPopup("")
		popupWidth := lipgloss.Width(popupStr)
		popupHeight := lipgloss.Height(popupStr)
		x := (m.width - popupWidth) / 2
		y := (m.height - popupHeight) / 2
		screen = PlaceOverlay(x, y, popupStr, screen, true)
	}

	if m.slashMode && len(m.slashMatches) > 0 {
		menuStr := m.renderSlashMenu("", mainWidth)
		menuWidth := lipgloss.Width(menuStr)
		menuHeight := lipgloss.Height(menuStr)
		x := (mainWidth - menuWidth) / 2
		inputHeight := lipgloss.Height(m.renderInput(mainWidth))
		y := m.height - inputHeight - menuHeight
		screen = PlaceOverlay(x, y, menuStr, screen, true)
	}

	return screen
}

func (m Model) renderMain(width int) string {
	inputArea := m.renderInput(width)
	msgHeight := m.height - lipgloss.Height(inputArea)

	content := m.renderMessages(width, msgHeight)
	full := lipgloss.JoinVertical(lipgloss.Bottom, content, inputArea)

	return lipgloss.NewStyle().
		Background(t.bg).
		Width(width).
		Height(m.height).
		Render(full)
}

func (m Model) renderMessages(width int, height int) string {
	content := m.viewport.View()
	if height < 3 {
		height = 3
	}

	return lipgloss.NewStyle().
		Background(t.bg).
		Width(width).
		Height(height).
		Padding(0, 1).
		Render(content)
}

func (m Model) renderInput(width int) string {
	inputWidth := width - 4
	if inputWidth < 10 {
		inputWidth = 10
	}

	// Meta line
	metaLeft := m.renderMetaLeft()
	metaRight := m.renderMetaRight(inputWidth)
	metaSpacer := strings.Repeat(" ", max(0, width-lipgloss.Width(metaLeft)-lipgloss.Width(metaRight)-2))
	metaLine := lipgloss.NewStyle().Padding(0, 1).Render(metaLeft + metaSpacer + metaRight)

	// OpenCode style borderless input
	prompt := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.primary).
		Padding(0, 0, 0, 1).
		Render(">")

	inputText := m.input.View()
	inputArea := lipgloss.JoinHorizontal(lipgloss.Top, prompt, " ", inputText)

	// Status line at very bottom
	statusLeft := m.renderStatusLeft()
	statusRight := dimStyle.Render("Tab sidebar · / commands · Esc cancel · Ctrl+C quit")
	statusSpacerLen := max(0, width-lipgloss.Width(statusLeft)-lipgloss.Width(statusRight)-4)
	statusSpacer := lipgloss.NewStyle().Background(t.bg).Render(strings.Repeat(" ", statusSpacerLen))
	statusLine := lipgloss.NewStyle().Background(t.bg).Padding(0, 1).Render(statusLeft + statusSpacer + statusRight)

	return lipgloss.JoinVertical(lipgloss.Left,
		metaLine,
		"", // spacing
		inputArea,
		"", // spacing
		statusLine,
	)
}

func (m Model) renderMetaLeft() string {
	provDef := provider.Get(m.cfg.Router.Provider)
	provLabel := m.cfg.Router.Provider
	if provDef != nil {
		provLabel = provDef.Label
	}
	modelLbl := provider.ModelLabel(m.cfg.Router.Provider, m.cfg.Router.Model)

	parts := []string{
		boldStyle.Foreground(t.accent).Render("Router"),
		mutedStyle.Render("·"),
		textStyle.Render(modelLbl),
		dimStyle.Render(provLabel),
	}

	if m.cfg.Router.ThinkingLevel != "" && m.cfg.Router.ThinkingLevel != "medium" {
		parts = append(parts, mutedStyle.Render("·"), warningStyle.Render("◆ "+m.cfg.Router.ThinkingLevel))
	}

	return strings.Join(parts, " ")
}

func (m Model) renderMetaRight(width int) string {
	if m.usageSnapshot == nil {
		return ""
	}
	if pu, ok := m.usageSnapshot.Providers[m.cfg.Router.Provider]; ok {
		if pu.Balance != "" {
			return successStyle.Render(pu.Balance + " left")
		}
		if pu.Error != "" && pu.Error != "no balance API" {
			return dimStyle.Render(pu.Error)
		}
	}
	return ""
}

func (m Model) renderStatusLeft() string {
	var status string
	switch m.state {
	case stateIdle:
		status = mutedStyle.Render("● Ready")
	case stateRouting:
		status = warningStyle.Render(m.spinner.View() + " Analyzing task...")
	case stateExecuting:
		active := len(m.activeTasks)
		status = lipgloss.NewStyle().Foreground(t.info).Render(fmt.Sprintf("● Executing %d task(s)", active))
	case stateDone:
		status = successStyle.Render("● Completed — type a new task")
	default:
		status = ""
	}

	if m.focusAgent != "" || m.focusTaskID >= 0 {
		status += "  " + lipgloss.NewStyle().Foreground(t.warning).Bold(true).Render("[ESC to return to main view]")
	}
	return status
}

func (m Model) renderSidebar() string {
	w := sidebarWidth
	var sb strings.Builder

	// Usage section
	sb.WriteString(boldStyle.Foreground(t.textMuted).Render(" USAGE") + "\n\n")

	if m.usageSnapshot != nil {
		if pu, ok := m.usageSnapshot.Providers[m.cfg.Router.Provider]; ok {
			if pu.Balance != "" {
				sb.WriteString("  " + successStyle.Render(pu.Balance+" left") + "\n")
			} else if pu.Error != "" && pu.Error != "no balance API" {
				sb.WriteString("  " + errorStyle.Render(pu.Error) + "\n")
			} else {
				sb.WriteString("  " + dimStyle.Render("Loading...") + "\n")
			}
		} else {
			sb.WriteString("  " + dimStyle.Render("Loading...") + "\n")
		}
	} else {
		sb.WriteString("  " + dimStyle.Render("Loading...") + "\n")
	}

	provDef := provider.Get(m.cfg.Router.Provider)
	modelLbl := m.cfg.Router.Model
	if provDef != nil {
		modelLbl = provider.ModelLabel(m.cfg.Router.Provider, m.cfg.Router.Model)
	}
	sb.WriteString("  " + dimStyle.Render(provDef.Label+" / "+modelLbl) + "\n")
	if m.cfg.Router.ThinkingLevel != "" {
		sb.WriteString("  " + dimStyle.Render("thinking: "+m.cfg.Router.ThinkingLevel) + "\n")
	}

	// Agents section
	agentsHdr := boldStyle.Foreground(t.textMuted).Render(" AGENTS")
	if m.sidebarFocus && m.sidebarMode == "agents" {
		agentsHdr = lipgloss.NewStyle().Foreground(t.text).Bold(true).Render("▶ AGENTS")
	} else if m.sidebarMode == "agents" {
		agentsHdr = boldStyle.Foreground(t.textMuted).Render("▶ AGENTS")
	}
	sb.WriteString("\n" + agentsHdr + "\n\n")

	agentOrder := []string{"claude", "codex", "opencode", "antigravity"}
	idx := 0
	for _, name := range agentOrder {
		if _, ok := m.agentStatus[name]; !ok {
			continue
		}
		selected := m.sidebarFocus && m.sidebarMode == "agents" && m.sidebarSelected == idx
		sb.WriteString(m.renderAgentUsageLine(name, selected))
		idx++
	}

	// Progress section
	sb.WriteString("\n" + boldStyle.Foreground(t.textMuted).Render(" PROGRESS") + "\n\n")
	sb.WriteString(m.renderProgress(w))

	tasksHdr := boldStyle.Foreground(t.textMuted).Render(" TASKS")
	if m.sidebarFocus && m.sidebarMode == "tasks" {
		tasksHdr = lipgloss.NewStyle().Foreground(t.text).Bold(true).Render("▶ TASKS")
	} else if m.sidebarMode == "tasks" {
		tasksHdr = boldStyle.Foreground(t.textMuted).Render("▶ TASKS")
	}
	sb.WriteString("\n\n" + tasksHdr + "\n\n")
	sb.WriteString(m.renderTaskList(w))

	content := sb.String()

	return lipgloss.NewStyle().
		Width(w).
		Height(m.height).
		Background(t.bgPanel).
		BorderLeft(true).
		BorderForeground(t.border).
		Padding(1, 1).
		Render(content)
}

func (m Model) renderAgentUsageLine(name string, selected bool) string {
	color := agentColor(name)
	label := agentLabel(name)
	status := m.agentStatus[name]

	dot := "○"
	dotColor := t.textDim
	switch status {
	case "running":
		dot = "●"
		dotColor = t.warning
	case "error":
		dot = "✗"
		dotColor = t.error
	case "queued":
		dot = "◐"
		dotColor = t.info
	}

	bg := t.bgPanel
	if selected {
		bg = t.bgElement
		nameLine := lipgloss.NewStyle().Background(bg).Foreground(t.text).Bold(true).Render("▶ ") +
			lipgloss.NewStyle().Background(bg).Foreground(dotColor).Render(dot) + " " +
			lipgloss.NewStyle().Background(bg).Foreground(color).Bold(true).Render(label)
		
		var renderedUsage string
		if m.usageSnapshot != nil {
			if u, ok := m.usageSnapshot.Agents[name]; ok {
				usageStr := u.StatusLine()
				barW := 10
				var filled int
				var barColor lipgloss.Color
				if u.HasPercent {
					filled = int(u.Percent * float64(barW))
					barColor = t.success
					if u.Percent < 0.2 { barColor = t.error }
				} else if u.Error == "" && u.LoggedIn {
					filled = barW
					barColor = t.info
				} else {
					filled = 0
					barColor = t.textDim
				}
				bar := lipgloss.NewStyle().Foreground(barColor).Background(bg).Render(strings.Repeat("█", filled)) +
					lipgloss.NewStyle().Foreground(t.textDim).Background(bg).Render(strings.Repeat("░", barW-filled))
				usageStr += "  " + bar

				if u.Error != "" {
					renderedUsage = lipgloss.NewStyle().Background(bg).Foreground(t.textDim).Render(usageStr)
				} else {
					renderedUsage = lipgloss.NewStyle().Background(bg).Foreground(t.textMuted).Render(usageStr)
				}
			}
		}
		if renderedUsage == "" {
			renderedUsage = lipgloss.NewStyle().Background(bg).Foreground(t.textDim).Render("...")
		}

		block := lipgloss.NewStyle().
			Background(bg).
			Width(sidebarWidth - 4). // account for padding
			Padding(0, 1).
			Render(nameLine + "\n  " + renderedUsage)
		
		return block + "\n\n"
	}

	nameLine := lipgloss.NewStyle().Foreground(dotColor).Render(dot) + " " +
		lipgloss.NewStyle().Foreground(color).Bold(true).Render(label)

	var renderedUsage string
	if m.usageSnapshot != nil {
		if u, ok := m.usageSnapshot.Agents[name]; ok {
			usageStr := u.StatusLine()
			barW := 10
			var filled int
			var barColor lipgloss.Color
			if u.HasPercent {
				filled = int(u.Percent * float64(barW))
				barColor = t.success
				if u.Percent < 0.2 { barColor = t.error }
			} else if u.Error == "" && u.LoggedIn {
				filled = barW
				barColor = t.info
			} else {
				filled = 0
				barColor = t.textDim
			}
			bar := lipgloss.NewStyle().Foreground(barColor).Render(strings.Repeat("█", filled)) +
				lipgloss.NewStyle().Foreground(t.textDim).Render(strings.Repeat("░", barW-filled))
			usageStr += "  " + bar

			if u.Error != "" {
				renderedUsage = lipgloss.NewStyle().Foreground(t.textDim).Render(usageStr)
			} else {
				renderedUsage = lipgloss.NewStyle().Foreground(t.textMuted).Render(usageStr)
			}
		}
	}
	if renderedUsage == "" {
		renderedUsage = lipgloss.NewStyle().Foreground(t.textDim).Render("...")
	}

	return nameLine + "\n  " + renderedUsage + "\n\n"
}

func (m Model) renderProgress(width int) string {
	if m.currentPlan == nil || len(m.currentPlan.Tasks) == 0 {
		return dimStyle.Render("  No active plan")
	}

	total := len(m.currentPlan.Tasks)
	done := len(m.completedTasks)
	pct := float64(done) / float64(total)

	barWidth := width - 12
	if barWidth < 5 {
		barWidth = 5
	}
	filled := int(pct * float64(barWidth))

	bar := lipgloss.NewStyle().Foreground(t.success).Render(strings.Repeat("█", filled)) +
		dimStyle.Render(strings.Repeat("░", barWidth-filled))

	return fmt.Sprintf("  %s %d/%d", bar, done, total)
}

func (m Model) renderTaskList(width int) string {
	if m.currentPlan == nil {
		return dimStyle.Render("  No tasks yet")
	}

	var sb strings.Builder
	for i, task := range m.currentPlan.Tasks {
		selected := m.sidebarFocus && m.sidebarMode == "tasks" && m.sidebarSelected == i

		status := "□"
		statusStyle := dimStyle
		if _, isActive := m.activeTasks[task.ID]; isActive {
			status = "▶"
			statusStyle = lipgloss.NewStyle().Foreground(t.warning)
		} else if m.completedTasks[task.ID] {
			status = "✓"
			statusStyle = lipgloss.NewStyle().Foreground(t.success)
		}

		color := agentColor(task.Agent)
		agentTag := lipgloss.NewStyle().Foreground(color).Render(agentLabel(task.Agent))

		desc := task.Description
		maxLen := width - 8
		if len(desc) > maxLen {
			if maxLen > 3 {
				desc = truncate.String(desc, uint(maxLen-3)) + "..."
			} else {
				desc = truncate.String(desc, uint(max(0, maxLen)))
			}
		}

		bg := t.bgPanel
		if selected {
			bg = t.bgElement
			agentTag = lipgloss.NewStyle().Background(bg).Foreground(color).Render(agentLabel(task.Agent))
			statusStyle = lipgloss.NewStyle().Background(bg).Foreground(statusStyle.GetForeground())
			desc = lipgloss.NewStyle().Background(bg).Foreground(t.textDim).Render(desc)
			sb.WriteString(lipgloss.NewStyle().Background(bg).Width(width-4).Padding(0, 1).Render(fmt.Sprintf("%s %s #%d %s\n  %s",
				lipgloss.NewStyle().Foreground(t.text).Bold(true).Render("▶"), statusStyle.Render(status), task.ID, agentTag, desc)) + "\n")
			continue
		}

		sb.WriteString(fmt.Sprintf("  %s #%d %s\n    %s\n",
			statusStyle.Render(status), task.ID, agentTag, dimStyle.Render(desc)))
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (m Model) renderPopup(background string) string {
	var content string
	var title string
	var popupWidth int
	var popupHeight int

	switch m.popup.kind {
	case popupHelp:
		title = "Keyboard Shortcuts"
		content = m.renderHelpContent()
		popupWidth = 58
		popupHeight = 14

	case popupModelPicker:
		title = "Select Router Model"
		content = m.renderModelPicker()
		popupWidth = 60
		popupHeight = 24

	case popupProviderManager:
		title = "Manage Providers"
		content = m.renderProviderManager()
		popupWidth = 62
		popupHeight = 24

	case popupThinkingPicker:
		title = "Thinking Level"
		content = m.renderThinkingPicker()
		popupWidth = 44
		popupHeight = 20

	case popupAgents:
		title = "Agents"
		content = m.renderAgentsPopup()
		popupWidth = 60
		popupHeight = 20

	case popupUsage:
		title = "Usage & Balance"
		content = m.renderUsagePopup()
		popupWidth = 60
		popupHeight = 26

	case popupCommandMenu:
		title = "Commands"
		content = m.renderCommandMenu(58)
		popupWidth = 60
		popupHeight = 14

	default:
		return background
	}

	popupWidth = max(10, min(popupWidth, m.width-4))
	popupHeight = max(5, min(popupHeight, m.height-4))

	header := lipgloss.NewStyle().
		Background(t.bgElement).
		Foreground(t.accent).
		Bold(true).
		Padding(0, 1).
		Render(" " + title + " ")

	hasInput := m.popup.kind == popupModelPicker || m.popup.kind == popupThinkingPicker
	if m.popup.kind == popupProviderManager && m.popup.addingProvider != "" {
		hasInput = true
	}
	inputLine := ""
	if hasInput {
		inputLine = "\n" + lipgloss.NewStyle().Foreground(t.info).Background(t.bgElement).Render("> ") +
			lipgloss.NewStyle().Background(t.bgElement).Render(m.popup.input) +
			lipgloss.NewStyle().Foreground(t.text).Background(t.bgElement).Render("▎")
	}

	body := content + inputLine
	bodyHeight := popupHeight - 3
	bodyLines := strings.Split(body, "\n")
	if len(bodyLines) > bodyHeight {
		bodyLines = bodyLines[len(bodyLines)-bodyHeight:]
		body = strings.Join(bodyLines, "\n")
	}

	innerBody := lipgloss.NewStyle().
		Background(t.bgElement).
		Padding(1, 1).
		Render(body)

	popup := lipgloss.NewStyle().
		Width(popupWidth).
		Height(popupHeight).
		Background(t.bgElement).
		BorderForeground(t.borderActive).
		Border(lipgloss.RoundedBorder()).
		Padding(0).
		Render(
			lipgloss.JoinVertical(lipgloss.Left, header, innerBody),
		)

	return popup
}

func (m Model) renderSlashMenu(background string, mainWidth int) string {
	menuWidth := min(50, mainWidth-4)
	innerWidth := menuWidth - 2
	var sb strings.Builder
	for i, cmd := range m.slashMatches {
		line := fmt.Sprintf(" /%-14s %s", cmd.name, cmd.description)
		pad := innerWidth - lipgloss.Width(line)
		if pad > 0 {
			line += strings.Repeat(" ", pad)
		} else if pad < 0 {
			if innerWidth > 3 {
				line = truncate.String(line, uint(innerWidth-3)) + "..."
			} else {
				line = truncate.String(line, uint(max(0, innerWidth)))
			}
		}

		if i == m.popup.selected {
			sb.WriteString(lipgloss.NewStyle().Background(t.accent).Foreground(lipgloss.Color("0")).Bold(true).Render(line))
		} else {
			sb.WriteString(lipgloss.NewStyle().Background(t.bgElement).Foreground(t.textMuted).Render(line))
		}
		sb.WriteString("\n")
	}


	menu := lipgloss.NewStyle().
		Width(menuWidth).
		Background(t.bgElement).
		BorderForeground(t.borderActive).
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Render(strings.TrimRight(sb.String(), "\n"))

	return menu
}

func (m Model) renderHelpContent() string {
	var sb strings.Builder
	shortcuts := [][2]string{
		{"/", "Show commands"},
		{"/model", "Choose router model"},
		{"/provider", "Add or manage API providers"},
		{"/thinking", "Set thinking level"},
		{"/agents", "Show all agents"},
		{"/usage", "Show usage & balance"},
		{"/clear", "Clear messages"},
		{"/help", "Show this help"},
		{"Enter", "Submit task"},
		{"Tab", "Toggle sidebar"},
		{"Esc", "Cancel / close popup"},
		{"PgUp/PgDn", "Scroll messages"},
		{"Ctrl+C", "Quit"},
	}
	for _, s := range shortcuts {
		key := lipgloss.NewStyle().Foreground(t.info).Render(s[0])
		desc := mutedStyle.Render(s[1])
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", key, desc))
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (m Model) renderThinkingPicker() string {
	var sb strings.Builder
	levels := []struct {
		id   string
		desc string
	}{
		{"low", "Fast and cheap — minimal analysis"},
		{"medium", "Balanced — default routing"},
		{"high", "Thorough — careful analysis"},
		{"max", "Exhaustive — maximum reasoning"},
	}
	for i, lv := range levels {
		var line string
		if m.popup.selected == i {
			markerStr := "  "
			if m.cfg.Router.ThinkingLevel == lv.id {
				markerStr = "→ "
			}
			currentStr := ""
			if m.cfg.Router.ThinkingLevel == lv.id {
				currentStr = " (current)"
			}
			
			line = fmt.Sprintf("%s%s%s", markerStr, lv.id, currentStr)
			pad := 38 - lipgloss.Width(line)
			if pad > 0 {
				line += strings.Repeat(" ", pad)
			}
			sb.WriteString(lipgloss.NewStyle().Background(t.accent).Foreground(lipgloss.Color("0")).Bold(true).Render(line) + "\n")
		} else {
			marker := "  "
			current := ""
			if m.cfg.Router.ThinkingLevel == lv.id {
				marker = lipgloss.NewStyle().Foreground(t.success).Render("→ ")
				current = lipgloss.NewStyle().Foreground(t.textDim).Render(" (current)")
			}
			label := boldStyle.Render(lv.id)
			line = fmt.Sprintf("%s%s%s", marker, label, current)
			sb.WriteString(line + "\n")
		}
		sb.WriteString(dimStyle.Render(lv.desc) + "\n\n")
	}
	sb.WriteString(dimStyle.Render("Press Enter to select or type level"))
	return sb.String()
}
