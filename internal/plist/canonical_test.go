package plist

import (
	"bytes"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/evnoj/launchdude/internal/config"
)

// TestTOMLEncoderDeterministic runs the BurntSushi/toml encoder 50 times on a
// struct containing a multi-key map. If the encoder iterates Go maps in random
// order (which is what Go's range does by default), some runs will differ —
// the test will fail and tell us we need to sort map keys ourselves.
func TestTOMLEncoderDeterministic(t *testing.T) {
	svc := &config.Service{
		Name:      "thing",
		Exec:      "/bin/foo --x --y",
		KeepAlive: true,
		RunAtLoad: true,
		Env: map[string]string{
			"ZULU":    "z",
			"ALPHA":   "a",
			"MIKE":    "m",
			"BRAVO":   "b",
			"OSCAR":   "o",
			"DELTA":   "d",
			"ECHO":    "e",
			"FOXTROT": "f",
		},
	}

	var first []byte
	for i := range 50 {
		var buf bytes.Buffer
		if err := toml.NewEncoder(&buf).Encode(svc); err != nil {
			t.Fatalf("encode run %d: %v", i, err)
		}
		got := buf.Bytes()
		if i == 0 {
			first = got
			t.Logf("encoder output on run 0:\n%s", string(got))
			continue
		}
		if !bytes.Equal(first, got) {
			t.Fatalf("run %d diverged from run 0\n--- run 0 ---\n%s\n--- run %d ---\n%s",
				i, string(first), i, string(got))
		}
	}
}

// TestHashConfigIgnoresMapOrder builds two Service values that are equivalent
// but populate the Env map in opposite insertion orders. HashConfig must
// produce the same hash for both — that's the whole point of canonicalizing.
func TestHashConfigIgnoresMapOrder(t *testing.T) {
	a := &config.Service{
		Name:      "x",
		Exec:      "/bin/foo",
		KeepAlive: true,
		Env:       map[string]string{},
	}
	b := &config.Service{
		Name:      "x",
		Exec:      "/bin/foo",
		KeepAlive: true,
		Env:       map[string]string{},
	}
	a.Env["X"] = "1"
	a.Env["Y"] = "2"
	a.Env["Z"] = "3"
	b.Env["Z"] = "3"
	b.Env["Y"] = "2"
	b.Env["X"] = "1"

	ha, hb := HashConfig(a), HashConfig(b)
	if ha != hb {
		t.Fatalf("hashes differ for equivalent configs\n  a: %s\n  b: %s", ha, hb)
	}
}

// TestHashConfigDetectsRealChange is the negative — actual behavioral changes
// must produce a different hash.
func TestHashConfigDetectsRealChange(t *testing.T) {
	a := &config.Service{Name: "x", Exec: "/bin/foo", KeepAlive: true}
	b := &config.Service{Name: "x", Exec: "/bin/bar", KeepAlive: true}
	if HashConfig(a) == HashConfig(b) {
		t.Fatal("hashes are equal despite different Exec")
	}
}
