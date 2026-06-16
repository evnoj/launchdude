package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/evnoj/launchdude/internal/service"
)

// PrintList writes a colored, aligned table of services to w.
// Columns: NAME, ENABLED, RUNNING, PID, INFO. ENABLED and RUNNING are
// independent axes — enabled = launchd is registered to know the service,
// running = there's a live process right now. INFO carries the special
// states (pending/orphan/modified) and last-exit info for failed runs.
func PrintList(w io.Writer, entries []*service.Status) {
	if len(entries) == 0 {
		fmt.Fprintln(w, Dim.Render("no services"))
		return
	}

	const cols = 5
	rows := make([][cols]string, 0, len(entries)+1)
	rows = append(rows, [cols]string{
		Bold.Render("NAME"),
		Bold.Render("ENABLED"),
		Bold.Render("RUNNING"),
		Bold.Render("PID"),
		Bold.Render("INFO"),
	})
	for _, st := range entries {
		rows = append(rows, [cols]string{
			st.Name,
			enabledListCell(st),
			runningListCell(st),
			pidCell(st),
			infoCell(st),
		})
	}

	widths := [cols]int{}
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

// enabledListCell answers: is launchd registered to manage this service?
// (Loaded == bootstrapped into the gui/<uid> domain.)
func enabledListCell(st *service.Status) string {
	if st.Loaded {
		return Running.Render("yes")
	}
	return Dim.Render("no")
}

// runningListCell answers: is there a live process right now? Failed runs
// (exit code non-zero, no current PID) get a red "no" to flag attention.
func runningListCell(st *service.Status) string {
	switch {
	case st.PID > 0:
		return Running.Render("yes")
	case st.Loaded && st.LastExitCode != 0:
		return Failed.Render("no")
	default:
		return Dim.Render("no")
	}
}

func pidCell(st *service.Status) string {
	if st.PID > 0 {
		return fmt.Sprintf("%d", st.PID)
	}
	return Dim.Render("-")
}

// infoCell aggregates the orthogonal status flags that don't fit the
// ENABLED/RUNNING axes: config<->plist relationship (pending/orphan/modified)
// and the last-exit-code for a service that's enabled but failing.
func infoCell(st *service.Status) string {
	var notes []string
	switch {
	case !st.ConfigExists && st.PlistExists:
		notes = append(notes, Orphan.Render("orphan"))
	case !st.PlistExists && st.ConfigExists:
		notes = append(notes, Pending.Render("pending"))
	}
	if st.Drifted {
		notes = append(notes, Modified.Render("modified"))
	}
	if st.Loaded && st.PID == 0 && st.LastExitCode != 0 {
		notes = append(notes, Failed.Render(fmt.Sprintf("exit %d", st.LastExitCode)))
	}
	return strings.Join(notes, " ")
}
