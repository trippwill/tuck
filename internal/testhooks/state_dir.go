package testhooks

import (
	"os"
	"path/filepath"
)

// StateHome returns the base machine-local state directory.
func StateHome() string {
	if override := testStateHomeOverride(); override != "" {
		return override
	}
	if xdgStateHome := os.Getenv("XDG_STATE_HOME"); xdgStateHome != "" {
		return xdgStateHome
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "state")
}

// TuckStateDir returns the directory that contains tuck's machine-local state.
func TuckStateDir() string {
	return filepath.Join(StateHome(), "tuck")
}

// SourcesFile returns the path to the machine-local source registry.
func SourcesFile() string {
	return filepath.Join(TuckStateDir(), "sources.toml")
}
