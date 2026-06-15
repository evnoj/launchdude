package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
)

const LabelPrefix = "launchdude."

func ConfigDir() (string, error) {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "launchdude"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "launchdude"), nil
}

func ServicesDir() (string, error) {
	cfg, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "services"), nil
}

func ServiceConfigPath(name string) (string, error) {
	dir, err := ServicesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".toml"), nil
}

func GlobalConfigPath() (string, error) {
	cfg, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "config.toml"), nil
}

func UserTemplatePath() (string, error) {
	cfg, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "template.toml"), nil
}

func LaunchAgentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents"), nil
}

func PlistPath(name string) (string, error) {
	dir, err := LaunchAgentsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, Label(name)+".plist"), nil
}

func LogsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Logs", "launchdude"), nil
}

func DefaultStdoutPath(name string) (string, error) {
	dir, err := LogsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".out.log"), nil
}

func DefaultStderrPath(name string) (string, error) {
	dir, err := LogsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".err.log"), nil
}

func Label(name string) string {
	return LabelPrefix + name
}

func NameFromLabel(label string) (string, bool) {
	if len(label) <= len(LabelPrefix) || label[:len(LabelPrefix)] != LabelPrefix {
		return "", false
	}
	return label[len(LabelPrefix):], true
}

func GUIDomain() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return "", fmt.Errorf("parse uid %q: %w", u.Uid, err)
	}
	return fmt.Sprintf("gui/%d", uid), nil
}

func ServiceTarget(name string) (string, error) {
	dom, err := GUIDomain()
	if err != nil {
		return "", err
	}
	return dom + "/" + Label(name), nil
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
