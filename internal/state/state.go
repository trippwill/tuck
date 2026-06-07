package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/trippwill/tuck/internal/apperr"
	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/testhooks"
)

type Source struct {
	ID       string
	Path     string
	Enabled  bool
	Manifest manifest.Manifest
}

type Registry struct {
	Sources []Source
	Default string
}

func (r Registry) EnabledSources() []Source {
	enabled := make([]Source, 0, len(r.Sources))
	for _, source := range r.Sources {
		if source.Enabled {
			enabled = append(enabled, source)
		}
	}
	return enabled
}

type Error = apperr.Error[ErrState]
type ErrState string

const ErrInvalid ErrState = "invalid state"

func (e ErrState) Error() string { return string(e) }

func Load() (Registry, error) {
	sourcesPath := testhooks.SourcesFile()
	contents, err := os.ReadFile(sourcesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Registry{}, nil
		}
		return Registry{}, apperr.Wrapf(ErrInvalid, err, "could not read state file %q", sourcesPath)
	}

	file := struct {
		Default string `toml:"default"`
		Sources []struct {
			ID      string `toml:"id"`
			Path    string `toml:"path"`
			Enabled *bool  `toml:"enabled"`
		} `toml:"source"`
	}{}

	if err := toml.Unmarshal(contents, &file); err != nil {
		return Registry{}, apperr.Wrapf(ErrInvalid, err, "could not parse state file %q", sourcesPath)
	}

	sources := make([]Source, len(file.Sources))
	enabledIDs := make(map[string]struct{})
	enabledPaths := make([]string, 0, len(file.Sources))
	for i, entry := range file.Sources {
		enabled := true
		if entry.Enabled != nil {
			enabled = *entry.Enabled
		}

		sources[i] = Source{
			ID:      entry.ID,
			Path:    entry.Path,
			Enabled: enabled,
		}

		if !enabled {
			continue
		}

		if !validID(entry.ID) {
			return Registry{}, apperr.Wrapf(ErrInvalid, nil, "invalid enabled source id %q: must be non-empty and cannot contain '/' or ':'", entry.ID)
		}
		if _, ok := enabledIDs[entry.ID]; ok {
			return Registry{}, apperr.Wrapf(ErrInvalid, nil, "duplicate enabled source id %q", entry.ID)
		}
		enabledIDs[entry.ID] = struct{}{}

		rootPath, err := canonicalRoot(entry.Path)
		if err != nil {
			return Registry{}, apperr.Wrapf(ErrInvalid, err, "invalid path for source %q", entry.ID)
		}
		if slices.ContainsFunc(enabledPaths, func(existing string) bool {
			return rootsOverlap(existing, rootPath)
		}) {
			return Registry{}, apperr.Wrapf(ErrInvalid, nil, "enabled source root %q overlaps another enabled source root", rootPath)
		}
		enabledPaths = append(enabledPaths, rootPath)

		sourceManifest, err := manifest.Load(rootPath)
		if err != nil {
			return Registry{}, apperr.Wrapf(ErrInvalid, err, "invalid manifest for source %q", entry.ID)
		}

		sources[i].Path = rootPath
		sources[i].Manifest = sourceManifest
	}

	if file.Default != "" {
		if _, ok := enabledIDs[file.Default]; !ok {
			return Registry{}, apperr.Wrapf(ErrInvalid, nil, "default source %q does not name an enabled source", file.Default)
		}
	}

	return Registry{
		Default: file.Default,
		Sources: sources,
	}, nil
}

func validID(id string) bool {
	return id != "" && !strings.ContainsAny(id, "/:")
}

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
