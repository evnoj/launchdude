package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TrashFile moves a file to ~/.Trash, matching Finder's collision-suffix
// convention ("foo.toml", "foo 2.toml", "foo 3.toml", ...). The file appears
// in the Trash UI like any user-deleted file (no "Put Back" support — that
// would require Finder automation, which prompts for permission).
//
// Same-volume only: ~/.Trash and ~/.config/launchdude both live in the user's
// home directory, so os.Rename is sufficient. Cross-volume support would need
// a copy-and-delete fallback for EXDEV.
func TrashFile(path string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	trashDir := filepath.Join(home, ".Trash")
	if err := os.MkdirAll(trashDir, 0o700); err != nil {
		return err
	}

	base := filepath.Base(path)
	dest := filepath.Join(trashDir, base)
	if _, err := os.Stat(dest); err == nil {
		dest = nextAvailable(trashDir, base)
	}
	if err := os.Rename(path, dest); err != nil {
		return fmt.Errorf("move %s to trash: %w", path, err)
	}
	return nil
}

// nextAvailable produces "stem N.ext" for the smallest N >= 2 that isn't taken.
// Matches Finder's convention so trashed files don't visually clash.
func nextAvailable(dir, base string) string {
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for i := 2; ; i++ {
		try := filepath.Join(dir, fmt.Sprintf("%s %d%s", stem, i, ext))
		if _, err := os.Stat(try); os.IsNotExist(err) {
			return try
		}
	}
}
