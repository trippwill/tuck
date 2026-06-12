package state

import (
	"os"
	"os/user"
	"path/filepath"
)

var (
	geteuid      = os.Geteuid
	lookupUserID = user.LookupId
)

func stateHome() string {
	if override := testStateHomeOverride(); override != "" {
		return override
	}
	if xdgStateHome := os.Getenv("XDG_STATE_HOME"); xdgStateHome != "" {
		return xdgStateHome
	}
	if sudoStateHome := invokingUserStateHome(); sudoStateHome != "" {
		return sudoStateHome
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "state")
}

func invokingUserStateHome() string {
	if geteuid() != 0 {
		return ""
	}
	sudoUID := os.Getenv("SUDO_UID")
	if sudoUID == "" {
		return ""
	}
	invokingUser, err := lookupUserID(sudoUID)
	if err != nil || invokingUser.HomeDir == "" {
		return ""
	}
	return filepath.Join(invokingUser.HomeDir, ".local", "state")
}

func tuckStateDir() string {
	return filepath.Join(stateHome(), "tuck")
}

func sourcesFile() string {
	return filepath.Join(tuckStateDir(), "sources.toml")
}
