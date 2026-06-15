package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrashFile_BasicMove(t *testing.T) {
	home := isolateHome(t)
	src := filepath.Join(home, "doomed.toml")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := TrashFile(src); err != nil {
		t.Fatalf("TrashFile: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source still exists after TrashFile: %v", err)
	}
	trashed := filepath.Join(home, ".Trash", "doomed.toml")
	if _, err := os.Stat(trashed); err != nil {
		t.Errorf("expected file in ~/.Trash, got %v", err)
	}
}

func TestTrashFile_CollisionSuffix(t *testing.T) {
	home := isolateHome(t)

	// Two files with the same basename, trashed in sequence.
	srcDirA := filepath.Join(home, "a")
	srcDirB := filepath.Join(home, "b")
	srcDirC := filepath.Join(home, "c")
	for _, d := range []string{srcDirA, srcDirB, srcDirC} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for i, d := range []string{srcDirA, srcDirB, srcDirC} {
		p := filepath.Join(d, "same.toml")
		if err := os.WriteFile(p, []byte("file"+string(rune('A'+i))), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := TrashFile(p); err != nil {
			t.Fatalf("trash run %d: %v", i, err)
		}
	}

	// Expect: same.toml, same 2.toml, same 3.toml in ~/.Trash
	for _, want := range []string{"same.toml", "same 2.toml", "same 3.toml"} {
		p := filepath.Join(home, ".Trash", want)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("missing trashed file %q: %v", want, err)
		}
	}
}

func TestTrashServiceConfig_IdempotentOnMissing(t *testing.T) {
	isolateHome(t)
	// No file exists; TrashServiceConfig should return nil (idempotent).
	if err := TrashServiceConfig("nothing-here"); err != nil {
		t.Errorf("expected nil on missing config, got %v", err)
	}
}

func TestTrashServiceConfig_MovesExisting(t *testing.T) {
	home := isolateHome(t)
	dir, _ := ServicesDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path, _ := ServiceConfigPath("doomed")
	if err := os.WriteFile(path, []byte(`name = "doomed"`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := TrashServiceConfig("doomed"); err != nil {
		t.Fatalf("TrashServiceConfig: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("config still at %s", path)
	}
	trashed := filepath.Join(home, ".Trash", "doomed.toml")
	if _, err := os.Stat(trashed); err != nil {
		t.Errorf("expected in trash: %v", err)
	}
}
