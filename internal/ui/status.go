package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/evnoj/launchdude/internal/config"
	"github.com/evnoj/launchdude/internal/service"
)

// PrintStatus writes a colored status block for a single service to w.
func PrintStatus(w io.Writer, st *service.Status) {
	label := config.Label(st.Name)
	pill := pillFor(st)

	fmt.Fprintf(w, "%s  %s\n", Bold.Render(label), pill)

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

	rows := [][2]string{}
	if st.Loaded {
		if st.PID > 0 {
			rows = append(rows, [2]string{"PID", fmt.Sprintf("%d", st.PID)})
		}
		if st.State != "" {
			rows = append(rows, [2]string{"State", st.State})
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

func pillFor(st *service.Status) string {
	switch {
	case !st.ConfigExists && st.PlistExists:
		return wrap(Orphan, "orphan")
	case !st.PlistExists && st.ConfigExists:
		return wrap(Pending, "pending")
	case st.Loaded && st.PID > 0:
		s := wrap(Running, "running")
		if st.Drifted {
			s += " " + wrap(Modified, "modified")
		}
		return s
	case st.Loaded && st.LastExitCode != 0:
		s := wrap(Failed, fmt.Sprintf("failed (exit %d)", st.LastExitCode))
		if st.Drifted {
			s += " " + wrap(Modified, "modified")
		}
		return s
	case st.Loaded:
		s := wrap(Stopped, "stopped")
		if st.Drifted {
			s += " " + wrap(Modified, "modified")
		}
		return s
	default:
		s := wrap(Stopped, "not loaded")
		if st.Drifted {
			s += " " + wrap(Modified, "modified")
		}
		return s
	}
}

func wrap(style interface{ Render(...string) string }, text string) string {
	return "[" + style.Render(text) + "]"
}
