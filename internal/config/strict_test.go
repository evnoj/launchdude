package config

import (
	"fmt"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestUndecodedReportsUnknownKeys(t *testing.T) {
	src := `
name = "foo"
exec = "/bin/true"
description = "this field was removed"
descriptoin = "typo"
`
	var svc Service
	md, err := toml.Decode(src, &svc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	und := md.Undecoded()
	t.Logf("Undecoded() returned %d keys: %v", len(und), und)
	for _, k := range und {
		t.Logf("  key: %s", k.String())
	}
	if len(und) == 0 {
		t.Fatal("expected Undecoded to flag the unknown keys, got 0")
	}
	// Confirm both unknown keys are reported.
	saw := map[string]bool{}
	for _, k := range und {
		saw[fmt.Sprint(k)] = true
	}
	if !saw["description"] || !saw["descriptoin"] {
		t.Fatalf("expected description and descriptoin in undecoded set, got %v", saw)
	}
}
