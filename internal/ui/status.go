package ui

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/evnoj/launchdude/internal/config"
	"github.com/evnoj/launchdude/internal/service"
)

// PrintStatus writes a colored status block for a single service to w.
// The pill row shows two independent axes — [enabled] and [running] — plus
// any drift/orphan/pending markers. Special states (pending, orphan) replace
// or augment the two-axis display where appropriate.
func PrintStatus(w io.Writer, st *service.Status) {
	printHeadline(w, st)
	printWarnings(w, st)
	printRows(w, statusRows(st))
}

// PrintShow writes the full status block (PrintStatus) plus the properties
// defined in the service's TOML config. svc may be nil (config missing or
// unparseable), in which case only the state block is printed.
func PrintShow(w io.Writer, st *service.Status, svc *config.Service) {
	printHeadline(w, st)
	printWarnings(w, st)
	printRows(w, append(statusRows(st), propertyRows(svc)...))
}

func printHeadline(w io.Writer, st *service.Status) {
	label := config.Label(st.Name)
	fmt.Fprintf(w, "%s  %s\n", Bold.Render(label), pillsFor(st))
}

func printWarnings(w io.Writer, st *service.Status) {
	if !st.ConfigExists && st.PlistExists {
		fmt.Fprintf(w, "  %s\n",
			Warn.Render("orphan: plist exists but no config; run `launchdude doctor` or `launchdude import "+st.Name+"`"))
	}
	if st.Drifted {
		fmt.Fprintf(w, "  %s\n",
			Warn.Render("config modified since last apply; run `launchdude apply "+st.Name+"` to reload"))
	}
	if !st.PlistExists && st.ConfigExists {
		fmt.Fprintf(w, "  %s\n",
			Dim.Render("plist not installed; run `launchdude enable "+st.Name+"` to install and start"))
	}
}

// statusRows builds the aligned key/value rows describing live state and the
// resolved on-disk paths.
func statusRows(st *service.Status) [][2]string {
	rows := [][2]string{}
	if st.Loaded {
		if st.PID > 0 {
			rows = append(rows, [2]string{"PID", fmt.Sprintf("%d", st.PID)})
		}
		if st.State != "" {
			rows = append(rows, [2]string{"launchd state", st.State})
		}
		rows = append(rows, [2]string{"Last exit", fmt.Sprintf("%d", st.LastExitCode)})
		if st.Program != "" {
			rows = append(rows, [2]string{"Program", st.Program})
		}
	}
	if st.ConfigExists {
		rows = append(rows, [2]string{"Config", st.ConfigPath})
	}
	if st.PlistExists {
		rows = append(rows, [2]string{"Plist", st.PlistPath})
	}
	if st.StdoutPath != "" {
		rows = append(rows, [2]string{"Stdout", st.StdoutPath})
	}
	if st.StderrPath != "" {
		rows = append(rows, [2]string{"Stderr", st.StderrPath})
	}
	if st.WorkingDir != "" {
		rows = append(rows, [2]string{"Working dir", st.WorkingDir})
	}
	return rows
}

// propertyRows builds the aligned key/value rows describing the properties
// defined in the service's TOML config. Returns nil when svc is nil.
func propertyRows(svc *config.Service) [][2]string {
	if svc == nil {
		return nil
	}
	var rows [][2]string
	if svc.Exec != "" {
		rows = append(rows, [2]string{"Exec", svc.Exec})
	}
	if len(svc.ExecArgs) > 0 {
		rows = append(rows, [2]string{"Args", strings.Join(svc.ExecArgs, " ")})
	}
	rows = append(rows, [2]string{"Keep alive", fmt.Sprintf("%t", svc.KeepAlive)})
	rows = append(rows, [2]string{"Run at load", fmt.Sprintf("%t", svc.RunAtLoad)})
	if len(svc.Env) > 0 {
		keys := make([]string, 0, len(svc.Env))
		for k := range svc.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = k + "=" + svc.Env[k]
		}
		rows = append(rows, [2]string{"Env", strings.Join(parts, " ")})
	}
	return rows
}

// printRows writes an aligned key/value block, padding keys to a common width.
func printRows(w io.Writer, rows [][2]string) {
	keyWidth := 0
	for _, r := range rows {
		if n := len(r[0]); n > keyWidth {
			keyWidth = n
		}
	}
	for _, r := range rows {
		pad := strings.Repeat(" ", keyWidth-len(r[0]))
		fmt.Fprintf(w, "  %s:%s  %s\n", Dim.Render(r[0]), pad, r[1])
	}
}

// pillsFor renders the headline pills for a service's status block. The two
// axes (enabled / running) are always shown when applicable. Special states
// (pending = config but no plist; orphan = plist but no config) take headline
// position; modified is shown as an additional pill alongside.
func pillsFor(st *service.Status) string {
	// Pending hides the two axes because they're both trivially "no":
	// there's no plist, so launchd can't know about it.
	if !st.PlistExists && st.ConfigExists {
		return wrap(Pending, "pending")
	}

	pills := []string{enabledPill(st), runningPill(st)}
	if !st.ConfigExists && st.PlistExists {
		// Orphan: the launchd state is real, so still show enabled/running,
		// but prepend the orphan flag so it stands out.
		pills = append([]string{wrap(Orphan, "orphan")}, pills...)
	}
	if st.Drifted {
		pills = append(pills, wrap(Modified, "modified"))
	}
	return strings.Join(pills, " ")
}

func enabledPill(st *service.Status) string {
	if st.Loaded {
		return wrap(Running, "enabled")
	}
	return wrap(Dim, "disabled")
}

func runningPill(st *service.Status) string {
	switch {
	case st.PID > 0:
		return wrap(Running, "running")
	case st.Loaded && st.LastExitCode != 0:
		return wrap(Failed, fmt.Sprintf("failed exit %d", st.LastExitCode))
	default:
		return wrap(Dim, "stopped")
	}
}

func wrap(style interface{ Render(...string) string }, text string) string {
	return "[" + style.Render(text) + "]"
}
