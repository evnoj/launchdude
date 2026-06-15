package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	Running  = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true) // bright green
	Stopped  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))            // yellow
	Failed   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)  // bright red
	Pending  = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))            // cyan
	Modified = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))            // magenta
	Orphan   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))             // red
	Dim      = lipgloss.NewStyle().Faint(true)
	Bold     = lipgloss.NewStyle().Bold(true)
	Warn     = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true) // yellow
)

// DisableColor forces lipgloss to render without ANSI codes. Used by --no-color
// and triggered automatically when NO_COLOR is set or stdout is not a TTY.
func DisableColor() {
	lipgloss.SetColorProfile(termenv.Ascii)
}
