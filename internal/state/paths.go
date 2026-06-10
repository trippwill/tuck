package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func canonicalRoot(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errEmptyPath
	}

	cleanPath := filepath.Clean(path)
	if cleanPath == "~" || strings.HasPrefix(cleanPath, "~"+string(os.PathSeparator)) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cleanPath = filepath.Join(home, strings.TrimPrefix(cleanPath, "~"))
	}
	if !filepath.IsAbs(cleanPath) {
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return "", err
		}
		cleanPath = absPath
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", cleanPath)
	}

	return filepath.EvalSymlinks(cleanPath)
}

func rootsOverlap(a, b string) bool {
	if a == b {
		return true
	}

	return isWithinRoot(a, b) || isWithinRoot(b, a)
}

func isWithinRoot(root, child string) bool {
	rel, err := filepath.Rel(root, child)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

var errEmptyPath = errors.New("source path is empty")
