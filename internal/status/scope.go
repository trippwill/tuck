package status

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/trippwill/tuck/internal/domain"
	"github.com/trippwill/tuck/internal/state"
)

func active(options Options) (state.Source, string, error) {
	targetRoot, err := domain.TargetRoot(options.TargetRoot, false)
	if err != nil {
		return state.Source{}, "", err
	}
	source, err := domain.ActiveSource(options.SourceID)
	if err != nil {
		return state.Source{}, "", err
	}
	return source, targetRoot, nil
}

func expandPath(raw string) (string, error) {
	path := raw
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = home
	} else if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		path = abs
	}
	return filepath.Clean(path), nil
}
