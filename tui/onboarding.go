package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"trainstation/config"
	"trainstation/provider"
)

type onboardingStep int

const (
	obStepProvider onboardingStep = iota
	obStepAPIKey
	obStepModel
	obStepAgentDetect
	obStepDone
)

type onboardingModel struct {
	cfg          *config.Config
	step         onboardingStep
	width        int
	height       int
	input        textarea.Model
	selectedIdx  int
	providerList []string
	filterText   string
	chosenProv   string
	errorMsg     string
	agents       map[string]bool
	skipPerms    bool
	fetchedModels []provider.ModelDef

	onComplete func(*config.Config)
}

func NewOnboarding(cfg *config.Config) tea.Model {
	return newOnboarding(cfg)
}

func newOnboarding(cfg *config.Config) onboardingModel {
	ta := textarea.New()
	styleTextArea(&ta)
	ta.Placeholder = ""
	ta.Focus()
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.SetHeight(1)

	provs := make([]string, 0, len(provider.Definitions))
	for _, def := range provider.Definitions {
		provs = append(provs, def.Name)
	}

	return onboardingModel{
		cfg:          cfg,
		step:         obStepProvider,
		input:        ta,
		providerList: provs,
		agents:       detectAgents(),
	}
}

func detectAgents() map[string]bool {
	result := map[string]bool{
		"claude":      false,
		"opencode":    false,
		"codex":       false,
		"antigravity": false,
	}
	for name := range result {
		if _, err := exec.LookPath(name); err == nil {
			result[name] = true
		}
	}
	return result
}

func (m onboardingModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m onboardingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(min(60, m.width-12))
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyEsc:
			if m.errorMsg != "" {
				m.errorMsg = ""
				return m, nil
			}
			if m.step == obStepAPIKey && m.chosenProv != "" {
				m.step = obStepProvider
				m.input.Reset()
				return m, nil
			}

		case tea.KeyUp:
			if m.step == obStepProvider && m.filterText == "" {
				if m.selectedIdx > 0 {
					m.selectedIdx--
				}
				return m, nil
			}
			if m.step == obStepModel {
				if m.selectedIdx > 0 {
					m.selectedIdx--
				}
				return m, nil
			}

		case tea.KeyDown:
			if m.step == obStepProvider && m.filterText == "" {
				if m.selectedIdx < len(m.providerList)-1 {
					m.selectedIdx++
				}
				return m, nil
			}
			if m.step == obStepModel {
				if m.selectedIdx < len(m.fetchedModels)-1 {
					m.selectedIdx++
				}
				return m, nil
			}

		case tea.KeyEnter:
			return m.handleEnter()

		case tea.KeyBackspace:
			if m.step == obStepProvider && m.filterText != "" {
				m.filterText = m.filterText[:len(m.filterText)-1]
				m.providerList = filterProviders(m.filterText)
				if m.selectedIdx >= len(m.providerList) {
					m.selectedIdx = max(0, len(m.providerList)-1)
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd

		default:
			if m.step == obStepProvider {
				if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
					m.filterText += string(msg.Runes)
					m.providerList = filterProviders(m.filterText)
					if m.selectedIdx >= len(m.providerList) {
						m.selectedIdx = max(0, len(m.providerList)-1)
					}
					return m, nil
				}
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m onboardingModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case obStepProvider:
		if len(m.providerList) == 0 {
			m.errorMsg = "No provider selected"
			return m, nil
		}
		if m.selectedIdx >= len(m.providerList) {
			m.selectedIdx = 0
		}
		m.chosenProv = m.providerList[m.selectedIdx]
		m.step = obStepAPIKey
		m.input.Reset()
		m.errorMsg = ""
		m.filterText = ""
		return m, nil

	case obStepAPIKey:
		key := strings.TrimSpace(m.input.Value())
		if key == "" {
			def := provider.Get(m.chosenProv)
			envVal := ""
			if def != nil {
				envVal = os.Getenv(def.EnvVar)
			}
			if envVal == "" {
				m.errorMsg = "Please enter an API key"
				return m, nil
			}
			key = envVal
		}
		m.cfg.SetProvider(m.chosenProv, key)
		m.cfg.Router.Provider = m.chosenProv
		m.errorMsg = ""

		def := provider.Get(m.chosenProv)
		if def != nil && def.ModelsPath != "" {
			// Fetch models dynamically from the API
			models := provider.GetModels(context.Background(), m.chosenProv, key)
			m.fetchedModels = models
			m.step = obStepModel
			m.input.Reset()
			m.selectedIdx = 0
		} else if def != nil && len(def.Models) > 0 {
			m.fetchedModels = def.Models
			m.step = obStepModel
			m.input.Reset()
			m.selectedIdx = 0
		} else {
			m.step = obStepAgentDetect
		}
		return m, nil

	case obStepModel:
		if m.selectedIdx >= len(m.fetchedModels) {
			m.step = obStepAgentDetect
			return m, nil
		}
		m.cfg.Router.Model = m.fetchedModels[m.selectedIdx].ID
		m.step = obStepAgentDetect
		return m, nil

	case obStepAgentDetect:
		m.cfg.Agents.Claude.Enabled = m.agents["claude"]
		m.cfg.Agents.OpenCode.Enabled = m.agents["opencode"]
		m.cfg.Agents.Codex.Enabled = m.agents["codex"]
		m.cfg.Agents.Antigravity.Enabled = m.agents["antigravity"]

		if m.skipPerms {
			m.cfg.Agents.Claude.SkipPermissions = true
			m.cfg.Agents.OpenCode.SkipPermissions = true
			m.cfg.Agents.Codex.SkipPermissions = true
			m.cfg.Agents.Antigravity.SkipPermissions = true
		}

		workspace, _ := os.Getwd()
		m.cfg.Workspace = workspace

		config.Save(m.cfg)
		m.step = obStepDone
		return m, tea.Quit
	}

	return m, nil
}

func filterProviders(prefix string) []string {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	var result []string
	for _, def := range provider.Definitions {
		if prefix == "" || strings.Contains(strings.ToLower(def.Name), prefix) || strings.Contains(strings.ToLower(def.Label), prefix) {
			result = append(result, def.Name)
		}
	}
	return result
}

func (m onboardingModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	logo := renderLogo(m.width, m.height/2+2)

	var content string
	switch m.step {
	case obStepProvider:
		content = m.renderProviderStep()
	case obStepAPIKey:
		content = m.renderAPIKeyStep()
	case obStepModel:
		content = m.renderModelStep()
	case obStepAgentDetect:
		content = m.renderAgentStep()
	}

	full := logo + "\n" + content

	return lipgloss.NewStyle().
		Background(t.bg).
		Width(m.width).
		Height(m.height).
		Padding(1, 2).
		Render(full)
}

func (m onboardingModel) renderProviderStep() string {
	var sb strings.Builder

	sb.WriteString(mutedStyle.Render("Select a provider for the router LLM:") + "\n\n")

	for i, name := range m.providerList {
		def := provider.Get(name)
		if def == nil {
			continue
		}
		envHint := ""
		if os.Getenv(def.EnvVar) != "" {
			envHint = successStyle.Render(" (env)")
		}
		if i == m.selectedIdx {
			line := fmt.Sprintf("  %s %s%s", lipgloss.NewStyle().Foreground(t.accent).Render("→"), def.Label, envHint)
			sb.WriteString(lipgloss.NewStyle().Background(t.bgElement).Render(line) + "\n")
		} else {
			sb.WriteString(fmt.Sprintf("    %s%s\n", def.Label, envHint))
		}
	}

	if m.filterText != "" {
		sb.WriteString("\n" + dimStyle.Render("Filter: "+m.filterText))
	}

	sb.WriteString("\n\n" + renderFoldedInput(m.input, t.accent, m.width, "Type to filter, ↑↓ to select, Enter to confirm"))

	return sb.String()
}

func (m onboardingModel) renderAPIKeyStep() string {
	var sb strings.Builder

	def := provider.Get(m.chosenProv)
	label := m.chosenProv
	if def != nil {
		label = def.Label
	}

	sb.WriteString(mutedStyle.Render("Enter API key for "+label+":") + "\n\n")

	envHint := ""
	if def != nil && os.Getenv(def.EnvVar) != "" {
		envHint = "\n" + dimStyle.Render("("+def.EnvVar+" found in env — press Enter to use it)")
	}

	if m.errorMsg != "" {
		sb.WriteString(errorStyle.Render("  ⚠ "+m.errorMsg) + "\n\n")
	}

	sb.WriteString(renderFoldedInput(m.input, t.accent, m.width, "Paste your API key..."))
	sb.WriteString(envHint)

	return sb.String()
}

func (m onboardingModel) renderModelStep() string {
	var sb strings.Builder

	def := provider.Get(m.chosenProv)
	if def == nil {
		return ""
	}

	sb.WriteString(mutedStyle.Render("Select a model for "+def.Label+":") + "\n\n")

	if len(m.fetchedModels) == 0 {
		sb.WriteString(dimStyle.Render("  Loading models...") + "\n")
		return sb.String()
	}

	for i, model := range m.fetchedModels {
		reasoner := ""
		if model.Reasoner {
			reasoner = warningStyle.Render(" ◆ reasoner")
		}
		if i == m.selectedIdx {
			line := fmt.Sprintf("  %s %s%s", lipgloss.NewStyle().Foreground(t.accent).Render("→"), model.Label, reasoner)
			sb.WriteString(lipgloss.NewStyle().Background(t.bgElement).Render(line) + "\n")
		} else {
			sb.WriteString(fmt.Sprintf("    %s%s\n", model.Label, reasoner))
		}
	}

	sb.WriteString("\n\n" + dimStyle.Render("↑↓ to select, Enter to confirm"))

	return sb.String()
}

func (m onboardingModel) renderAgentStep() string {
	var sb strings.Builder

	sb.WriteString(mutedStyle.Render("Detected agents:") + "\n\n")

	agentOrder := []string{"claude", "codex", "opencode", "antigravity"}
	for _, name := range agentOrder {
		label := agentLabel(name)
		color := agentColor(name)
		if m.agents[name] {
			sb.WriteString(fmt.Sprintf("  %s %s\n", successStyle.Render("✓"), lipgloss.NewStyle().Foreground(color).Render(label)))
		} else {
			sb.WriteString(fmt.Sprintf("  %s %s\n", dimStyle.Render("✗"), dimStyle.Render(label)))
		}
	}

	sb.WriteString("\n" + renderFoldedInput(m.input, t.borderActive, m.width, "Press Enter to finish setup"))

	return sb.String()
}

// renderFoldedInput creates a modern OpenCode-style prompt input
func renderFoldedInput(input textarea.Model, color lipgloss.Color, width int, placeholder string) string {
	inputText := input.View()
	if inputText == "" || strings.TrimSpace(inputText) == "" {
		inputText = lipgloss.NewStyle().Foreground(t.textDim).Render(placeholder)
	}

	prompt := lipgloss.NewStyle().
		Bold(true).
		Foreground(color).
		Padding(0, 0, 0, 1).
		Render(">")

	return lipgloss.JoinHorizontal(lipgloss.Top, prompt, " ", inputText)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
