package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Global struct {
	DefaultTemplate string `toml:"default_template,omitempty"`
}

func LoadGlobal() (*Global, error) {
	path, err := GlobalConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Global{}, nil
	}
	if err != nil {
		return nil, err
	}
	var g Global
	if _, err := toml.Decode(string(data), &g); err != nil {
		return nil, err
	}
	return &g, nil
}
