package gui

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Config is the on-disk user-preferences blob persisted between
// sessions. It's intentionally tiny and JSON-serialized so the operator
// can `cat` and edit it if they need to.
//
// Path:
//
//	macOS:   ~/Library/Application Support/DumpSock/config.json
//	Linux:   ${XDG_CONFIG_HOME:-~/.config}/dumpsock/config.json
//	Windows: %APPDATA%/DumpSock/config.json
type Config struct {
	// Version is bumped whenever the schema changes incompatibly.
	Version int `json:"version"`

	// LastOutput is the folder the user last picked via the GUI's "Choose…"
	// button. Empty on first launch.
	LastOutput string `json:"last_output,omitempty"`
}

const currentConfigVersion = 1

// configPath returns the absolute path where the config blob lives.
// Uses os.UserConfigDir for platform-correct location.
func configPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("UserConfigDir: %w", err)
	}
	return filepath.Join(base, "DumpSock", "config.json"), nil
}

// loadConfig reads the config file. A missing file is not an error —
// returns a zero-valued Config so first-launch is silent.
func loadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Config{Version: currentConfigVersion}, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		// Corrupt config: start fresh rather than crash the GUI.
		return Config{Version: currentConfigVersion}, nil
	}
	if c.Version == 0 {
		c.Version = currentConfigVersion
	}
	return c, nil
}

// saveConfig writes the config atomically (tmp + rename) so a crash or
// power loss mid-write can't leave the file half-written.
func saveConfig(c Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if c.Version == 0 {
		c.Version = currentConfigVersion
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir config dir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write tmp config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename tmp config: %w", err)
	}
	return nil
}
