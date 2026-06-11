package domain

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/trippwill/tuck/internal/resolve"
	"github.com/trippwill/tuck/internal/state"
)

var ErrNoHome = errors.New("HOME is not set")

type TargetScope struct {
	Context      string
	LogicalRoot  string
	PhysicalRoot string
}

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

func NewTargetScope(context string, explicit string, requireHome bool) (TargetScope, error) {
	if context == "root" {
		physicalRoot := explicit
		if physicalRoot == "" {
			physicalRoot = rootPhysicalRoot()
		}
		return TargetScope{
			Context:      "root",
			LogicalRoot:  string(filepath.Separator),
			PhysicalRoot: filepath.Clean(physicalRoot),
		}, nil
	}

	targetRoot, err := TargetRoot(explicit, requireHome)
	if err != nil {
		return TargetScope{}, err
	}
	return TargetScope{
		Context:      "home",
		LogicalRoot:  targetRoot,
		PhysicalRoot: targetRoot,
	}, nil
}

func (s TargetScope) PhysicalPath(logicalPath string) string {
	logicalPath = filepath.Clean(logicalPath)
	if s.Context != "root" {
		return logicalPath
	}
	if logicalPath == string(filepath.Separator) {
		return s.PhysicalRoot
	}
	rel, err := filepath.Rel(string(filepath.Separator), logicalPath)
	if err != nil || rel == "." || rel == ".." || rel == "" {
		return s.PhysicalRoot
	}
	return filepath.Clean(filepath.Join(s.PhysicalRoot, rel))
}
