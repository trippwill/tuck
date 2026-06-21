package state

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/pelletier/go-toml/v2"
	"github.com/trippwill/tuck/internal/apperr"
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
	Copies  []Copy
}

type Copy struct {
	Source         string
	Context        string
	Package        string
	Path           string
	Target         string
	SourceChecksum string
	TargetChecksum string
	TargetMode     string
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

type ErrState string

func (e ErrState) Error() string { return string(e) }

const (
	ErrInvalid          ErrState = "invalid state"
	ErrChecksumMismatch ErrState = "state checksum mismatch"
	ErrSourceRoot       ErrState = "source root"
	ErrWrite            ErrState = "state write"
)

func Load() (Registry, error) {
	sourcesPath := sourcesFile()
	contents, err := os.ReadFile(sourcesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Registry{}, nil
		}
		return Registry{}, apperr.AppErrWrapf(ErrInvalid, err, "could not read state file %q", sourcesPath)
	}
	if err := validateStateChecksum(contents); err != nil {
		return Registry{}, err
	}

	file := struct {
		Default string `toml:"default"`
		Sources []struct {
			ID      string `toml:"id"`
			Path    string `toml:"path"`
			Enabled *bool  `toml:"enabled"`
		} `toml:"source"`
		Copies []struct {
			Source         string `toml:"source"`
			Context        string `toml:"context"`
			Package        string `toml:"package"`
			Path           string `toml:"path"`
			Target         string `toml:"target"`
			SourceChecksum string `toml:"sourceChecksum"`
			TargetChecksum string `toml:"targetChecksum"`
			TargetMode     string `toml:"targetMode"`
		} `toml:"copy"`
	}{}

	if err := toml.Unmarshal(contents, &file); err != nil {
		return Registry{}, apperr.AppErrWrapf(ErrInvalid, err, "could not parse state file %q", sourcesPath)
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

	copies := make([]Copy, len(file.Copies))
	for i, entry := range file.Copies {
		copies[i] = Copy{
			Source:         entry.Source,
			Context:        entry.Context,
			Package:        entry.Package,
			Path:           entry.Path,
			Target:         entry.Target,
			SourceChecksum: entry.SourceChecksum,
			TargetChecksum: entry.TargetChecksum,
			TargetMode:     entry.TargetMode,
		}
	}

	return normalizeRegistry(Registry{
		Default: file.Default,
		Sources: sources,
		Copies:  copies,
	})
}

func validateStateChecksum(contents []byte) error {
	checksumPath := sourcesChecksumFile()
	recorded, err := os.ReadFile(checksumPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return apperr.AppErrWrapf(ErrChecksumMismatch, err, "could not read state checksum sidecar %q", checksumPath)
	}
	if got, want := strings.TrimSpace(string(recorded)), stateChecksum(contents); got != want {
		return apperr.AppErrMsgf(ErrChecksumMismatch, "machine source state checksum does not match sources.toml")
	}
	return nil
}

func stateChecksum(contents []byte) string {
	sum := sha256.Sum256(contents)
	return fmt.Sprintf("sha256:%x", sum)
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
		return apperr.AppErrWrapf(ErrWrite, err, "could not create state directory %q", stateDir)
	}

	tempPath, err := writeStateTemp(stateDir, ".sources.toml.*", contents)
	if err != nil {
		return err
	}
	defer os.Remove(tempPath)
	checksumTempPath, err := writeStateTemp(stateDir, ".sources.toml.sha256.*", []byte(stateChecksum(contents)+"\n"))
	if err != nil {
		return err
	}
	defer os.Remove(checksumTempPath)
	if err := os.Rename(tempPath, sourcesPath); err != nil {
		return apperr.AppErrWrapf(ErrWrite, err, "could not replace state file %q", sourcesPath)
	}
	if err := os.Rename(checksumTempPath, sourcesChecksumFile()); err != nil {
		return apperr.AppErrWrapf(ErrWrite, err, "could not replace state checksum sidecar %q", sourcesChecksumFile())
	}
	return nil
}

func writeStateTemp(stateDir string, pattern string, contents []byte) (string, error) {
	tempFile, err := os.CreateTemp(stateDir, pattern)
	if err != nil {
		return "", apperr.AppErrWrapf(ErrWrite, err, "could not create temporary state file in %q", stateDir)
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
		return "", apperr.AppErrWrapf(ErrWrite, err, "could not write temporary state file %q", tempPath)
	}
	if err := tempFile.Close(); err != nil {
		return "", apperr.AppErrWrapf(ErrWrite, err, "could not close temporary state file %q", tempPath)
	}
	if err := chownStateTemp(tempPath, stateDir); err != nil {
		return "", err
	}
	removeTemp = false
	return tempPath, nil
}

func chownStateTemp(tempPath string, stateDir string) error {
	if geteuid() != 0 {
		return nil
	}
	info, err := os.Stat(stateDir)
	if err != nil {
		return apperr.AppErrWrapf(ErrWrite, err, "could not inspect state directory %q", stateDir)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	if err := os.Chown(tempPath, int(stat.Uid), int(stat.Gid)); err != nil {
		return apperr.AppErrWrapf(ErrWrite, err, "could not set state file owner %q", tempPath)
	}
	return nil
}

func AddSource(path string, makeDefault bool) (Registry, Source, error) {
	rootPath, err := canonicalRoot(path)
	if err != nil {
		return Registry{}, Source{}, apperr.AppErrWrapf(ErrSourceRoot, err, "invalid source root %q", path)
	}
	sourceManifest, err := manifest.Load(rootPath)
	if err != nil {
		return Registry{}, Source{}, err
	}

	added := Source{
		ID:       sourceManifest.Name,
		Path:     rootPath,
		Enabled:  true,
		Manifest: sourceManifest,
	}

	normalized, _, err := mutateRegistry(func(registry Registry) (Registry, bool, error) {
		found := false
		for i, existing := range registry.Sources {
			if existing.ID != sourceManifest.Name {
				continue
			}
			existingPath, err := canonicalRoot(existing.Path)
			if err != nil {
				return Registry{}, false, apperr.AppErrWrapf(ErrInvalid, err, "existing source %q has invalid path %q", existing.ID, existing.Path)
			}
			if existingPath != rootPath {
				return Registry{}, false, apperr.AppErrMsgf(ErrInvalid, "source id %q already exists at %q", existing.ID, existingPath)
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
		return registry, true, nil
	})
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
	var removed Source
	normalized, changed, err := mutateRegistry(func(registry Registry) (Registry, bool, error) {
		removedIndex := -1
		for i, source := range registry.Sources {
			if source.ID != id {
				continue
			}
			removedIndex = i
			removed = source
			break
		}
		if removedIndex == -1 {
			return registry, false, nil
		}

		registry.Sources = append(registry.Sources[:removedIndex], registry.Sources[removedIndex+1:]...)
		if registry.Default == id {
			registry.Default = ""
		}
		return registry, true, nil
	})
	if err != nil {
		return Registry{}, Source{}, false, err
	}
	if !changed {
		return normalized, Source{}, false, nil
	}
	return normalized, removed, true, nil
}

func SetDefault(id string) (Registry, Source, bool, error) {
	var selected Source
	normalized, changed, err := mutateRegistry(func(registry Registry) (Registry, bool, error) {
		for _, source := range registry.Sources {
			if source.ID != id || !source.Enabled {
				continue
			}
			selected = source
			registry.Default = id
			return registry, true, nil
		}
		return registry, false, nil
	})
	if err != nil {
		return Registry{}, Source{}, false, err
	}
	if !changed {
		return normalized, Source{}, false, nil
	}
	return normalized, findSource(normalized, selected.ID), true, nil
}

func mutateRegistry(mutate func(Registry) (Registry, bool, error)) (Registry, bool, error) {
	registry, err := Load()
	if err != nil {
		return Registry{}, false, err
	}
	updated, changed, err := mutate(registry)
	if err != nil {
		return Registry{}, false, err
	}
	if !changed {
		return registry, false, nil
	}
	if err := Save(updated); err != nil {
		return Registry{}, false, err
	}
	normalized, err := Load()
	if err != nil {
		return Registry{}, false, err
	}
	return normalized, true, nil
}

func findSource(registry Registry, id string) Source {
	for _, source := range registry.Sources {
		if source.ID == id {
			return source
		}
	}
	return Source{}
}
