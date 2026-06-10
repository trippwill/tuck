package state

import (
	"os"
	"path/filepath"
)

func stateHome() string {
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

func tuckStateDir() string {
	return filepath.Join(stateHome(), "tuck")
}

func sourcesFile() string {
	return filepath.Join(tuckStateDir(), "sources.toml")
}
