package domain

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/trippwill/tuck/internal/resolve"
	"github.com/trippwill/tuck/internal/state"
)

var ErrNoHome = errors.New("HOME is not set")

func ActiveSource(sourceID string) (state.Source, error) {
	registry, err := state.Load()
	if err != nil {
		return state.Source{}, err
	}
	return resolve.ActiveSource(registry, sourceID)
}

func TargetRoot(explicit string, requireHome bool) (string, error) {
	targetRoot := explicit
	if targetRoot == "" {
		targetRoot = os.Getenv("HOME")
	}
	if targetRoot == "" {
		if requireHome {
			return "", ErrNoHome
		}
		targetRoot = "."
	}
	return filepath.Clean(targetRoot), nil
}
