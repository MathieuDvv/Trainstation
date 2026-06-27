package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

type theme struct {
	bg           lipgloss.Color
	bgDim        lipgloss.Color
	bgPanel      lipgloss.Color
	bgElement    lipgloss.Color
	bgHover      lipgloss.Color
	border       lipgloss.Color
	borderActive lipgloss.Color
	text         lipgloss.Color
	textMuted    lipgloss.Color
	textDim      lipgloss.Color
	primary      lipgloss.Color
	secondary    lipgloss.Color
	accent       lipgloss.Color
	success      lipgloss.Color
	warning      lipgloss.Color
	error        lipgloss.Color
	info         lipgloss.Color

	claude      lipgloss.Color
	codex       lipgloss.Color
	opencode    lipgloss.Color
	antigravity lipgloss.Color
	router      lipgloss.Color
}

var t = theme{
	// Transparent/Terminal background to prevent grey bars
	bg:           lipgloss.Color(""),
	bgDim:        lipgloss.Color(""),
	bgPanel:      lipgloss.Color("#111111"), // subtle panels
	bgElement:    lipgloss.Color("#1A1A1A"), // subtle elements
	bgHover:      lipgloss.Color("#222222"),
	border:       lipgloss.Color("#333333"),
	borderActive: lipgloss.Color("#555555"),
	text:         lipgloss.Color("#EEEEEE"),
	textMuted:    lipgloss.Color("#888888"),
	textDim:      lipgloss.Color("#555555"),
	primary:      lipgloss.Color("#00E5FF"), // Neon cyan
	secondary:    lipgloss.Color("#B026FF"), // Neon purple
	accent:       lipgloss.Color("#00E5FF"),
	success:      lipgloss.Color("#00FF88"),
	warning:      lipgloss.Color("#FFD500"),
	error:        lipgloss.Color("#FF3366"),
	info:         lipgloss.Color("#00E5FF"),

	claude:      lipgloss.Color("#FF8800"),
	codex:       lipgloss.Color("#00FFAA"),
	opencode:    lipgloss.Color("#00E5FF"),
	antigravity: lipgloss.Color("#B026FF"),
	router:      lipgloss.Color("#FFD500"),
}

func agentColor(name string) lipgloss.Color {
	switch name {
	case "claude":
		return t.claude
	case "codex":
		return t.codex
	case "opencode":
		return t.opencode
	case "antigravity":
		return t.antigravity
	case "router":
		return t.router
	case "error":
		return t.error
	default:
		return t.text
	}
}

func agentLabel(name string) string {
	switch name {
	case "claude":
		return "Claude Code"
	case "codex":
		return "Codex"
	case "opencode":
		return "OpenCode"
	case "antigravity":
		return "Antigravity"
	case "router":
		return "Router"
	default:
		return name
	}
}

var (
	sidebarWidth = 38

	textStyle    = lipgloss.NewStyle().Foreground(t.text)
	mutedStyle   = lipgloss.NewStyle().Foreground(t.textMuted)
	dimStyle     = lipgloss.NewStyle().Foreground(t.textDim)
	boldStyle    = lipgloss.NewStyle().Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(t.error)
	successStyle = lipgloss.NewStyle().Foreground(t.success)
	warningStyle = lipgloss.NewStyle().Foreground(t.warning)
	infoStyle    = lipgloss.NewStyle().Foreground(t.info)

	// OpenCode-style thick left border for messages
	leftBar = func(color lipgloss.Color) string {
		return lipgloss.NewStyle().Foreground(color).Render("┃ ")
	}
)

// inputBorderColor returns the color for the input left border based on state
func inputBorderColor(state appState) lipgloss.Color {
	switch state {
	case stateRouting:
		return t.warning
	case stateExecuting:
		return t.info
	case stateDone:
		return t.success
	default:
		return t.borderActive
	}
}

// styleTextArea removes default grey backgrounds from text inputs
func styleTextArea(ta *textarea.Model) {
	bg := lipgloss.NewStyle().Background(t.bg)
	ta.FocusedStyle.Base = bg
	ta.FocusedStyle.CursorLine = bg
	ta.FocusedStyle.Prompt = bg
	ta.FocusedStyle.Text = bg.Foreground(t.text)
	ta.FocusedStyle.Placeholder = bg.Foreground(t.textDim)
	
	ta.BlurredStyle.Base = bg
	ta.BlurredStyle.CursorLine = bg
	ta.BlurredStyle.Prompt = bg
	ta.BlurredStyle.Text = bg.Foreground(t.text)
	ta.BlurredStyle.Placeholder = bg.Foreground(t.textDim)
	
	// Remove the default prompt because we render our own
	ta.Prompt = ""
}
