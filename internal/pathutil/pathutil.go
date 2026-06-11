package pathutil

import (
	"os"
	"path/filepath"
	"strings"
)

type pathErr string

func (e pathErr) Error() string { return string(e) }

const errPathEscape pathErr = "path escapes root"

func Inside(child, root string) bool {
	child = filepath.Clean(child)
	root = filepath.Clean(root)
	if child == root {
		return true
	}
	rel, err := filepath.Rel(root, child)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func RelInside(root, child string) (string, error) {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(child))
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errPathEscape
	}
	return rel, nil
}

func PackageToTarget(packageRoot, entryPath, targetRoot string) (string, string, error) {
	rel, err := RelInside(packageRoot, entryPath)
	if err != nil {
		return "", "", err
	}
	targetPath := filepath.Clean(filepath.Join(targetRoot, rel))
	if !Inside(targetPath, targetRoot) {
		return "", "", errPathEscape
	}
	return targetPath, rel, nil
}

func TargetToPackage(targetRoot, targetPath, packageRoot string) (string, string, error) {
	rel, err := RelInside(targetRoot, targetPath)
	if err != nil {
		return "", "", err
	}
	packagePath := filepath.Clean(filepath.Join(packageRoot, rel))
	if !Inside(packagePath, packageRoot) {
		return "", "", errPathEscape
	}
	return packagePath, rel, nil
}

func SymlinkPayload(linkPath, targetPath string) (string, error) {
	return filepath.Rel(filepath.Dir(linkPath), targetPath)
}

func ExpandInput(raw string) (string, error) {
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
