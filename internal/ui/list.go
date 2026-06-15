package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/evnoj/launchdude/internal/service"
)

// PrintList writes a colored, aligned table of services to w.
// Columns: NAME, STATUS, PID, NOTES. ANSI-aware column widths via lipgloss.Width.
func PrintList(w io.Writer, entries []*service.Status) {
	if len(entries) == 0 {
		fmt.Fprintln(w, Dim.Render("no services"))
		return
	}

	rows := make([][4]string, 0, len(entries)+1)
	rows = append(rows, [4]string{
		Bold.Render("NAME"),
		Bold.Render("STATUS"),
		Bold.Render("PID"),
		Bold.Render("NOTES"),
	})
	for _, st := range entries {
		rows = append(rows, [4]string{
			st.Name,
			listStatusCell(st),
			pidCell(st),
			notesCell(st),
		})
	}

	widths := [4]int{}
	for _, r := range rows {
		for i, cell := range r {
			if cw := lipgloss.Width(cell); cw > widths[i] {
				widths[i] = cw
			}
		}
	}

	for _, r := range rows {
		var line strings.Builder
		for i, cell := range r {
			line.WriteString(cell)
			if i < len(r)-1 {
				line.WriteString(strings.Repeat(" ", widths[i]-lipgloss.Width(cell)+2))
			}
		}
		fmt.Fprintln(w, strings.TrimRight(line.String(), " "))
	}
}

func listStatusCell(st *service.Status) string {
	switch {
	case !st.ConfigExists && st.PlistExists:
		return Orphan.Render("orphan")
	case !st.PlistExists && st.ConfigExists:
		return Pending.Render("pending")
	case st.Loaded && st.PID > 0:
		return Running.Render("running")
	case st.Loaded && st.LastExitCode != 0:
		return Failed.Render(fmt.Sprintf("failed (%d)", st.LastExitCode))
	case st.Loaded:
		return Stopped.Render("stopped")
	default:
		return Stopped.Render("not loaded")
	}
}

func pidCell(st *service.Status) string {
	if st.PID > 0 {
		return fmt.Sprintf("%d", st.PID)
	}
	return Dim.Render("-")
}

func notesCell(st *service.Status) string {
	var notes []string
	if st.Drifted {
		notes = append(notes, Modified.Render("modified"))
	}
	if !st.ConfigExists && st.PlistExists {
		notes = append(notes, Dim.Render("no config"))
	}
	if !st.PlistExists && st.ConfigExists {
		notes = append(notes, Dim.Render("no plist"))
	}
	return strings.Join(notes, " ")
}
