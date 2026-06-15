package ui

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Prompt reads a line from r and returns its first non-whitespace character
// (lowercased). Returns 0 if the line is empty or on read error.
func Prompt(r *bufio.Reader, w io.Writer, msg string) byte {
	fmt.Fprint(w, msg)
	line, err := r.ReadString('\n')
	if err != nil {
		return 0
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return 0
	}
	return line[0]
}

// Confirm asks a yes/no question; default is no.
func Confirm(r *bufio.Reader, w io.Writer, msg string) bool {
	c := Prompt(r, w, msg+" [y/N] ")
	return c == 'y'
}
