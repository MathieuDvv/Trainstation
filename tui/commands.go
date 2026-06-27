package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/atotto/clipboard"

	"trainstation/config"
	"trainstation/provider"
)

type commandDef struct {
	name        string
	description string
}

var commandList = []commandDef{
	{"model", "Choose router model"},
	{"provider", "Add or manage API providers"},
	{"thinking", "Set thinking level (low/medium/high/max)"},
	{"agents", "Show all agents and their status"},
	{"usage", "Show detailed usage and balance"},
	{"clear", "Clear message history"},
	{"focus", "Focus on an agent's logs (e.g. /focus claude)"},
	{"copy", "Copy all logs to clipboard"},
	{"help", "Show keyboard shortcuts"},
}

func (m *Model) handleSlashCommand(input string) (handled bool, err error) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return false, nil
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return true, nil
	}

	cmdName := strings.TrimPrefix(parts[0], "/")

	switch cmdName {
	case "model":
		var opts []string
		for _, provName := range m.cfg.ConfiguredProviders() {
			apiKey := m.cfg.GetAPIKey(provName)
			models := provider.GetModels(m.ctx, provName, apiKey)
			for _, mod := range models {
				opts = append(opts, provName+":"+mod.ID)
			}
		}
		m.popup = popupModel{kind: popupModelPicker, options: opts}
		// pre-select current
		for i, o := range opts {
			if o == m.cfg.Router.Provider+":"+m.cfg.Router.Model {
				m.popup.selected = i
			}
		}
		return true, nil

	case "provider":
		m.popup = popupModel{kind: popupProviderManager}
		if len(m.cfg.ConfiguredProviders()) == 0 {
			m.popup.provSection = 1
		}
		return true, nil

	case "thinking":
		if len(parts) >= 2 {
			return true, m.setThinkingLevel(parts[1])
		}
		opts := []string{"low", "medium", "high", "max"}
		m.popup = popupModel{kind: popupThinkingPicker, options: opts}
		for i, o := range opts {
			if o == m.cfg.Router.ThinkingLevel {
				m.popup.selected = i
			}
		}
		return true, nil

	case "agents":
		m.popup = popupModel{kind: popupAgents}
		return true, nil

	case "usage":
		m.popup = popupModel{kind: popupUsage}
		return true, nil

	case "clear":
		m.entries = nil
		m.addInfoEntry("Messages cleared.")
		m.refreshViewport()
		return true, nil

	case "help":
		m.popup = popupModel{kind: popupHelp}
		return true, nil

	case "focus":
		if len(parts) >= 2 {
			m.focusAgent = parts[1]
			m.resize()
			m.refreshViewport()
			return true, nil
		}
		return true, fmt.Errorf("usage: /focus <agent_name>")

	case "copy":
		var sb strings.Builder
		for _, e := range m.entries {
			sb.WriteString(e.text.String())
			sb.WriteString("\n")
		}
		err := clipboard.WriteAll(sb.String())
		if err != nil {
			return true, fmt.Errorf("failed to copy to clipboard: %v", err)
		}
		m.addInfoEntry("Copied all logs to clipboard!")
		return true, nil

	case "panic":
		panic("Testing crash handler")

	default:
		return true, fmt.Errorf("unknown command: /%s — try /help", cmdName)
	}
}

func (m *Model) setThinkingLevel(level string) error {
	valid := map[string]bool{"low": true, "medium": true, "high": true, "max": true}
	if !valid[level] {
		return fmt.Errorf("invalid thinking level: %s (use low, medium, high, or max)", level)
	}
	m.cfg.Router.ThinkingLevel = level
	if m.router != nil {
		m.router.SetThinking(level)
	}
	config.Save(m.cfg)
	m.addInfoEntry(fmt.Sprintf("Thinking level set to: %s", level))
	return nil
}

func (m *Model) getMatchingCommands(prefix string) []commandDef {
	prefix = strings.TrimPrefix(prefix, "/")
	var matches []commandDef
	for _, cmd := range commandList {
		if strings.HasPrefix(cmd.name, prefix) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

func (m *Model) renderCommandMenu(width int) string {
	var sb strings.Builder
	sb.WriteString(boldStyle.Foreground(t.accent).Render("Commands") + "\n\n")
	for _, cmd := range commandList {
		name := infoStyle.Render("/" + cmd.name)
		desc := mutedStyle.Render(cmd.description)
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", name, desc))
	}
	return sb.String()
}

func (m *Model) renderModelPicker() string {
	var sb strings.Builder

	configured := m.cfg.ConfiguredProviders()
	if len(configured) == 0 {
		sb.WriteString(mutedStyle.Render("No providers configured. Use /provider to add one."))
		return sb.String()
	}

	idx := 0
	for _, provName := range configured {
		def := provider.Get(provName)
		if def == nil {
			continue
		}
		color := agentColor("router")
		header := boldStyle.Foreground(color).Render(def.Label)
		balance := ""
		if m.usageSnapshot != nil {
			if pu, ok := m.usageSnapshot.Providers[provName]; ok && pu.Balance != "" {
				balance = successStyle.Render("  " + pu.Balance + " left")
			}
		}
		sb.WriteString(header + balance + "\n")

		// Get dynamic models (from API cache or fallback to hardcoded)
		apiKey := m.cfg.GetAPIKey(provName)
		models := provider.GetModels(m.ctx, provName, apiKey)

		if len(models) == 0 {
			sb.WriteString("  " + dimStyle.Render("No models available") + "\n\n")
			continue
		}

		for _, model := range models {
			marker := "  "
			current := ""
			if m.cfg.Router.Provider == provName && m.cfg.Router.Model == model.ID {
				marker = successStyle.Render("→ ")
				current = dimStyle.Render(" (current)")
			}
			
			label := model.Label
			if m.popup.selected == idx {
				label = lipgloss.NewStyle().Background(t.accent).Foreground(t.bg).Bold(true).Padding(0, 1).Render(label)
			}
			idx++

			reasoner := ""
			if model.Reasoner {
				reasoner = warningStyle.Render(" ◆")
			}
			sb.WriteString(fmt.Sprintf("%s%s%s%s\n", marker, label, reasoner, current))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(dimStyle.Render("Press Enter to select or type provider:model"))
	return sb.String()
}

func (m *Model) renderProviderManager() string {
	var sb strings.Builder

	if m.popup.addingProvider != "" {
		def := provider.Get(m.popup.addingProvider)
		label := m.popup.addingProvider
		if def != nil {
			label = def.Label
		}
		sb.WriteString(boldStyle.Foreground(t.accent).Render("Add " + label) + "\n\n")
		sb.WriteString(mutedStyle.Render("Enter API key:") + "\n\n")
		if m.popup.input != "" {
			masked := strings.Repeat("*", len(m.popup.input))
			sb.WriteString("  " + textStyle.Render(masked) + "\n")
		} else {
			sb.WriteString("  " + dimStyle.Render("(type or press Enter for env var)") + "\n")
		}
		envHint := ""
		if def != nil {
			if ev := os.Getenv(def.EnvVar); ev != "" {
				envHint = dimStyle.Render("(" + def.EnvVar + " found in env)")
			}
		}
		if envHint != "" {
			sb.WriteString("\n  " + envHint)
		}
		sb.WriteString("\n\n" + dimStyle.Render("Enter to confirm · Esc to cancel"))
		return sb.String()
	}

	configured := m.cfg.ConfiguredProviders()
	availSection := mutedStyle
	cfgSection := mutedStyle
	if m.popup.provSection == 0 {
		cfgSection = lipgloss.NewStyle().Foreground(t.accent).Bold(true)
	} else {
		availSection = lipgloss.NewStyle().Foreground(t.accent).Bold(true)
	}

	sb.WriteString(cfgSection.Render("▸ Configured") + "  " + availSection.Render("▸ Available") + "\n\n")

	if m.popup.provSection == 0 {
		if len(configured) == 0 {
			sb.WriteString("  " + dimStyle.Render("none") + "\n")
		}
		for i, name := range configured {
			selected := m.popup.provSelected == i
			marker := "  "
			var bg lipgloss.Color
			if selected {
				marker = lipgloss.NewStyle().Foreground(t.error).Render("✕ ")
				bg = t.bgHover
			}
			def := provider.Get(name)
			label := name
			if def != nil {
				label = def.Label
			}
			key := m.cfg.GetAPIKey(name)
			masked := ""
			if len(key) > 8 {
				masked = key[:4] + "..." + key[len(key)-4:]
			} else if len(key) > 0 {
				masked = "***"
			}
			line := fmt.Sprintf("%s%s  %s", marker, boldStyle.Render(label), dimStyle.Render(masked))
			if selected {
				line = lipgloss.NewStyle().Background(bg).Render(line)
			}
			sb.WriteString(line + "\n")
		}
		sb.WriteString("\n" + dimStyle.Render("Enter to remove selected"))
	} else {
		available := m.availableProviders()
		if len(available) == 0 {
			sb.WriteString("  " + dimStyle.Render("all providers configured") + "\n")
		}
		for i, name := range available {
			selected := m.popup.provSelected == i
			marker := "  "
			var bg lipgloss.Color
			if selected {
				marker = lipgloss.NewStyle().Foreground(t.success).Render("+ ")
				bg = t.bgHover
			}
			def := provider.Get(name)
			label := name
			if def != nil {
				label = def.Label
			}
			envHint := ""
			if def != nil {
				if ev := os.Getenv(def.EnvVar); ev != "" {
					envHint = successStyle.Render(" (env)")
				}
			}
			line := fmt.Sprintf("%s%s%s", marker, label, envHint)
			if selected {
				line = lipgloss.NewStyle().Background(bg).Render(line)
			}
			sb.WriteString(line + "\n")
		}
		sb.WriteString("\n" + dimStyle.Render("Enter to add selected"))
	}

	sb.WriteString("\n\n" + dimStyle.Render("Tab switch · ↑↓ select · Esc close"))
	return sb.String()
}

func (m *Model) renderAgentsPopup() string {
	var sb strings.Builder
	sb.WriteString(boldStyle.Foreground(t.accent).Render("Agents") + "\n\n")

	agentOrder := []string{"claude", "codex", "opencode", "antigravity"}
	for _, name := range agentOrder {
		cfg := getAgentConfig(m.cfg, name)
		if cfg == nil || !cfg.Enabled {
			continue
		}

		color := agentColor(name)
		label := agentLabel(name)
		sb.WriteString(boldStyle.Foreground(color).Render(label) + "\n")

		if m.usageSnapshot != nil {
			if u, ok := m.usageSnapshot.Agents[name]; ok {
				status := u.StatusLine()
				if u.Error != "" {
					sb.WriteString("  " + errorStyle.Render(status) + "\n")
				} else {
					sb.WriteString("  " + mutedStyle.Render(status) + "\n")
				}
			}
		}

		sb.WriteString("  " + dimStyle.Render("strengths: "+strings.Join(cfg.Strengths, ", ")) + "\n\n")
	}

	return sb.String()
}

func (m *Model) renderUsagePopup() string {
	var sb strings.Builder
	sb.WriteString(boldStyle.Foreground(t.accent).Render("Usage & Balance") + "\n\n")

	sb.WriteString(boldStyle.Foreground(t.textMuted).Render("Router Provider") + "\n")
	if m.usageSnapshot != nil {
		if pu, ok := m.usageSnapshot.Providers[m.cfg.Router.Provider]; ok {
			if pu.Balance != "" {
				sb.WriteString("  " + successStyle.Render(pu.Balance+" remaining") + "\n")
			} else if pu.Error != "" {
				sb.WriteString("  " + errorStyle.Render(pu.Error) + "\n")
			}
		}
	}
	sb.WriteString("\n")

	sb.WriteString(boldStyle.Foreground(t.textMuted).Render("Agents") + "\n")
	agentOrder := []string{"claude", "codex", "opencode", "antigravity"}
	for _, name := range agentOrder {
		if m.usageSnapshot == nil {
			continue
		}
		if u, ok := m.usageSnapshot.Agents[name]; ok {
			color := agentColor(name)
			label := agentLabel(name)
			status := u.StatusLine()
			if u.HasPercent {
				barW := 15
				filled := int(u.Percent * float64(barW))
				bar := lipgloss.NewStyle().Foreground(t.success).Render(strings.Repeat("█", filled)) +
					lipgloss.NewStyle().Foreground(t.textDim).Render(strings.Repeat("░", barW-filled))
				status += "  " + bar
			}
			sb.WriteString(lipgloss.NewStyle().Foreground(color).Render(label) + "  " + mutedStyle.Render(status) + "\n")
		}
	}

	return sb.String()
}
