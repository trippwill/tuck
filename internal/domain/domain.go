package domain

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/trippwill/tuck/internal/resolve"
	"github.com/trippwill/tuck/internal/state"
)

var ErrNoHome = errors.New("HOME is not set")

const (
	ContextHome = "home"
	ContextRoot = "root"
)

type TargetScope struct {
	Context      string
	LogicalRoot  string
	PhysicalRoot string
}

type SelectionOptions struct {
	SourceID    string
	Context     string
	TargetRoot  string
	RequireHome bool
}

type ActiveSelection struct {
	Registry state.Registry
	Source   state.Source
	Scope    TargetScope
}

func ActiveSource(sourceID string) (state.Source, error) {
	registry, err := state.Load()
	if err != nil {
		return state.Source{}, err
	}
	return resolve.ActiveSource(registry, sourceID)
}

func SelectActive(options SelectionOptions) (ActiveSelection, error) {
	scope, err := NewTargetScope(options.Context, options.TargetRoot, options.RequireHome)
	if err != nil {
		return ActiveSelection{}, err
	}
	registry, err := state.Load()
	if err != nil {
		return ActiveSelection{}, err
	}
	source, err := resolve.ActiveSource(registry, options.SourceID)
	if err != nil {
		return ActiveSelection{}, err
	}
	return ActiveSelection{
		Registry: registry,
		Source:   source,
		Scope:    scope,
	}, nil
}

func NormalizeContext(context string) string {
	if context == ContextRoot {
		return ContextRoot
	}
	return ContextHome
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
	context = NormalizeContext(context)
	if context == ContextRoot {
		physicalRoot := explicit
		if physicalRoot == "" {
			physicalRoot = rootPhysicalRoot()
		}
		return TargetScope{
			Context:      ContextRoot,
			LogicalRoot:  string(filepath.Separator),
			PhysicalRoot: filepath.Clean(physicalRoot),
		}, nil
	}

	targetRoot, err := TargetRoot(explicit, requireHome)
	if err != nil {
		return TargetScope{}, err
	}
	return TargetScope{
		Context:      ContextHome,
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
