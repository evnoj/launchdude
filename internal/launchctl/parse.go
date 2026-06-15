package launchctl

import (
	"bufio"
	"strconv"
	"strings"
)

// parsePrint extracts the fields we care about from `launchctl print` output.
// The format is curly-brace nested key/value (not JSON; not stable across macOS
// versions), so we only pluck the few keys we use and ignore the rest.
//
// Example input (abridged):
//
//	gui/501/launchdude.foo = {
//		active count = 0
//		path = /Users/evan/Library/LaunchAgents/launchdude.foo.plist
//		type = LaunchAgent
//		program = /usr/local/bin/foo
//		state = running
//		pid = 12345
//		last exit code = 0
//		arguments = {
//			/usr/local/bin/foo
//		}
//	}
func parsePrint(output string) *ServiceState {
	st := &ServiceState{Loaded: true}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		eq := strings.Index(line, "=")
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		// Skip nested-block openers like "arguments = {".
		if val == "{" {
			continue
		}
		switch key {
		case "path":
			st.PlistPath = val
		case "program":
			st.Program = val
		case "state":
			st.State = val
		case "pid":
			if n, err := strconv.Atoi(val); err == nil {
				st.PID = n
			}
		case "last exit code":
			if n, err := strconv.Atoi(val); err == nil {
				st.LastExitCode = n
			}
		}
	}
	return st
}

// parseList parses `launchctl list` tab-separated rows. First line is the header.
//
//	PID   Status   Label
//	12345 0        launchdude.foo
//	-     0        com.apple.something
func parseList(output, prefix string) []ListEntry {
	var entries []ListEntry
	scanner := bufio.NewScanner(strings.NewReader(output))
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		label := fields[2]
		if prefix != "" && !strings.HasPrefix(label, prefix) {
			continue
		}
		e := ListEntry{Label: label}
		if fields[0] != "-" {
			if n, err := strconv.Atoi(fields[0]); err == nil {
				e.PID = n
			}
		}
		if n, err := strconv.Atoi(fields[1]); err == nil {
			e.Status = n
		}
		entries = append(entries, e)
	}
	return entries
}
