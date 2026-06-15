package config

import (
	_ "embed"
	"os"
	"strings"
)

//go:embed default_template.toml
var defaultTemplate string

// Template returns the contents for a new service's TOML config.
// Precedence: user's global default_template path (if set and readable),
// then ~/.config/launchdude/template.toml, then the embedded default.
// {{NAME}} placeholders are replaced with the service name.
func Template(name string) (string, error) {
	if g, err := LoadGlobal(); err == nil && g.DefaultTemplate != "" {
		expanded, err := expandUser(g.DefaultTemplate)
		if err == nil {
			if data, err := os.ReadFile(expanded); err == nil {
				return render(string(data), name), nil
			}
		}
	}
	if utp, err := UserTemplatePath(); err == nil {
		if data, err := os.ReadFile(utp); err == nil {
			return render(string(data), name), nil
		}
	}
	return render(defaultTemplate, name), nil
}

func render(tmpl, name string) string {
	return strings.ReplaceAll(tmpl, "{{NAME}}", name)
}
