// Package config loads dotr's own settings from ~/.config/dotr/config.yaml.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const fileName = "config.yaml"

// Config is dotr application settings.
type Config struct {
	BackupKeep     int      `yaml:"backup_keep"`
	ChromaStyle    string   `yaml:"chroma_style"`
	ConfirmSecrets bool     `yaml:"confirm_secrets"`
	Mouse          bool     `yaml:"mouse"`
	Watch          bool     `yaml:"watch"`
	GitStatus      bool     `yaml:"git_status"`
	StowDir        string   `yaml:"stow_dir"`
	StowTarget     string   `yaml:"stow_target"`
	ExtraIgnores   []string `yaml:"extra_ignores"`
}

// Default returns sensible defaults.
func Default() Config {
	return Config{
		BackupKeep:     20,
		ChromaStyle:    "dracula",
		ConfirmSecrets: true,
		Mouse:          true,
		Watch:          true,
		GitStatus:      true,
	}
}

// Dir returns ~/.config/dotr (respecting XDG_CONFIG_HOME).
func Dir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "dotr"), nil
}

// Path returns the config file path.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

// Load reads config.yaml. Missing file yields defaults.
func Load() (Config, error) {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Default(), err
	}
	cfg = cfg.withDefaults()
	return cfg, nil
}

func (c Config) withDefaults() Config {
	d := Default()
	if c.BackupKeep <= 0 {
		c.BackupKeep = d.BackupKeep
	}
	if c.ChromaStyle == "" {
		c.ChromaStyle = d.ChromaStyle
	}
	// ConfirmSecrets and Mouse: zero value false is valid, but we want
	// missing keys to mean true. Track via pointer in a private unmarshal
	// would be ideal; for simplicity, if file exists and mouse is omitted
	// yaml gives false. Document defaults in EnsureFile instead.
	return c
}

// EnsureFile writes a default config if none exists.
func EnsureFile() (string, error) {
	path, err := Path()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	const body = `# dotr config
backup_keep: 20
chroma_style: dracula
confirm_secrets: true
mouse: true
watch: true
git_status: true

# GNU Stow (empty = read ~/.stowrc)
# stow_dir: ~/repos
# stow_target: ~

# Extra ignore patterns (same syntax as ignore file)
# extra_ignores:
#   - "**/skills/"
#   - cursor/
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return "", err
	}
	return path, nil
}
