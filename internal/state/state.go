package state

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/trippwill/tuck/internal/manifest"
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

//go:generate go run ../../cmd/errgen -type ErrState
type ErrState string

const (
	ErrInvalid    ErrState = "invalid state"
	ErrSourceRoot ErrState = "source root"
	ErrWrite      ErrState = "state write"
)

func Load() (Registry, error) {
	sourcesPath := sourcesFile()
	contents, err := os.ReadFile(sourcesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Registry{}, nil
		}
		return Registry{}, AppErrWrapf(ErrInvalid, err, "could not read state file %q", sourcesPath)
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
		return Registry{}, AppErrWrapf(ErrInvalid, err, "could not parse state file %q", sourcesPath)
	}

	sources := make([]Source, len(file.Sources))
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
	}

	return normalizeRegistry(Registry{
		Default: file.Default,
		Sources: sources,
	})
}

func Save(registry Registry) error {
	normalized, err := normalizeRegistry(registry)
	if err != nil {
		return err
	}
	contents := marshalRegistry(normalized)
	sourcesPath := sourcesFile()
	stateDir := filepath.Dir(sourcesPath)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return AppErrWrapf(ErrWrite, err, "could not create state directory %q", stateDir)
	}

	tempFile, err := os.CreateTemp(stateDir, ".sources.toml.*")
	if err != nil {
		return AppErrWrapf(ErrWrite, err, "could not create temporary state file in %q", stateDir)
	}
	tempPath := tempFile.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := tempFile.Write(contents); err != nil {
		_ = tempFile.Close()
		return AppErrWrapf(ErrWrite, err, "could not write temporary state file %q", tempPath)
	}
	if err := tempFile.Close(); err != nil {
		return AppErrWrapf(ErrWrite, err, "could not close temporary state file %q", tempPath)
	}
	if err := os.Rename(tempPath, sourcesPath); err != nil {
		return AppErrWrapf(ErrWrite, err, "could not replace state file %q", sourcesPath)
	}
	removeTemp = false
	return nil
}

func AddSource(path string, makeDefault bool) (Registry, Source, error) {
	rootPath, err := canonicalRoot(path)
	if err != nil {
		return Registry{}, Source{}, AppErrWrapf(ErrSourceRoot, err, "invalid source root %q", path)
	}
	sourceManifest, err := manifest.Load(rootPath)
	if err != nil {
		return Registry{}, Source{}, err
	}

	registry, err := Load()
	if err != nil {
		return Registry{}, Source{}, err
	}

	added := Source{
		ID:       sourceManifest.Name,
		Path:     rootPath,
		Enabled:  true,
		Manifest: sourceManifest,
	}

	found := false
	for i, existing := range registry.Sources {
		if existing.ID != sourceManifest.Name {
			continue
		}
		existingPath, err := canonicalRoot(existing.Path)
		if err != nil {
			return Registry{}, Source{}, AppErrWrapf(ErrInvalid, err, "existing source %q has invalid path %q", existing.ID, existing.Path)
		}
		if existingPath != rootPath {
			return Registry{}, Source{}, AppErrMsgf(ErrInvalid, "source id %q already exists at %q", existing.ID, existingPath)
		}
		registry.Sources[i] = added
		found = true
		break
	}
	if !found {
		registry.Sources = append(registry.Sources, added)
	}
	if makeDefault {
		registry.Default = added.ID
	}

	if err := Save(registry); err != nil {
		return Registry{}, Source{}, err
	}
	normalized, err := Load()
	if err != nil {
		return Registry{}, Source{}, err
	}
	return normalized, findSource(normalized, added.ID), nil
}

func AddSourceWithInit(path string, makeDefault bool, initOptions manifest.InitOptions) (Registry, Source, error) {
	registry, source, err := AddSource(path, makeDefault)
	if err == nil {
		return registry, source, nil
	}
	if !errors.Is(err, manifest.ErrMissing) && !errors.Is(err, ErrSourceRoot) {
		return Registry{}, Source{}, err
	}
	initialized, initErr := manifest.Init(path, initOptions)
	if initErr != nil {
		return Registry{}, Source{}, initErr
	}
	return AddSource(initialized.Root, makeDefault)
}

func RemoveSource(id string) (Registry, Source, bool, error) {
	registry, err := Load()
	if err != nil {
		return Registry{}, Source{}, false, err
	}

	removedIndex := -1
	var removed Source
	for i, source := range registry.Sources {
		if source.ID != id {
			continue
		}
		removedIndex = i
		removed = source
		break
	}
	if removedIndex == -1 {
		return registry, Source{}, false, nil
	}

	registry.Sources = append(registry.Sources[:removedIndex], registry.Sources[removedIndex+1:]...)
	if registry.Default == id {
		registry.Default = ""
	}
	if err := Save(registry); err != nil {
		return Registry{}, Source{}, false, err
	}
	normalized, err := Load()
	if err != nil {
		return Registry{}, Source{}, false, err
	}
	return normalized, removed, true, nil
}

func findSource(registry Registry, id string) Source {
	for _, source := range registry.Sources {
		if source.ID == id {
			return source
		}
	}
	return Source{}
}
