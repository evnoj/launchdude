package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRequireArgs_MissingOneNamesTheArg(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	err := requireArgs("NAME")(cmd, []string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "requires NAME" {
		t.Errorf("error: %q", err.Error())
	}
	// The error must carry the *cobra.Command so the top-level handler can
	// print its help; that's the whole point of the custom type.
	var mae *MissingArgsError
	if !errors.As(err, &mae) {
		t.Fatalf("expected *MissingArgsError, got %T", err)
	}
	if mae.Cmd != cmd {
		t.Error("MissingArgsError.Cmd should reference the offending command")
	}
}

func TestRequireArgs_MissingMultipleListsAll(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	err := requireArgs("FROM", "TO")(cmd, []string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "requires FROM TO" {
		t.Errorf("error: %q", err.Error())
	}
}

func TestRequireArgs_PartiallyGivenListsRemaining(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	err := requireArgs("FROM", "TO")(cmd, []string{"a"})
	if err == nil {
		t.Fatal("expected error")
	}
	// Only TO is missing; the validator should say so.
	if !strings.Contains(err.Error(), "TO") {
		t.Errorf("expected error to mention TO: %q", err.Error())
	}
	if strings.Contains(err.Error(), "FROM") {
		t.Errorf("FROM was supplied; should not appear in error: %q", err.Error())
	}
}

func TestRequireArgs_TooManyReturnsExtraArgsError(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	err := requireArgs("NAME")(cmd, []string{"a", "b"})
	if err == nil {
		t.Fatal("expected error")
	}
	var eae *ExtraArgsError
	if !errors.As(err, &eae) {
		t.Fatalf("expected *ExtraArgsError, got %T (%v)", err, err)
	}
	if !strings.Contains(err.Error(), "NAME") {
		t.Errorf("error should mention the expected arg(s): %q", err.Error())
	}
}

func TestRequireArgs_ExactCountIsHappy(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	if err := requireArgs("NAME")(cmd, []string{"foo"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
