package state

import (
	"slices"
	"strings"

	"github.com/trippwill/tuck/internal/apperr"
	"github.com/trippwill/tuck/internal/manifest"
)

func normalizeRegistry(registry Registry) (Registry, error) {
	normalized := Registry{
		Default: registry.Default,
		Sources: make([]Source, len(registry.Sources)),
	}
	copy(normalized.Sources, registry.Sources)

	enabledIDs := make(map[string]struct{})
	enabledPaths := make([]string, 0, len(normalized.Sources))
	for i, source := range normalized.Sources {
		if !source.Enabled {
			continue
		}

		if !validID(source.ID) {
			return Registry{}, apperr.AppErrMsgf(ErrInvalid, "invalid enabled source id %q: must be non-empty and cannot contain '/' or ':'", source.ID)
		}
		if _, ok := enabledIDs[source.ID]; ok {
			return Registry{}, apperr.AppErrMsgf(ErrInvalid, "duplicate enabled source id %q", source.ID)
		}
		enabledIDs[source.ID] = struct{}{}

		rootPath, err := canonicalRoot(source.Path)
		if err != nil {
			return Registry{}, apperr.AppErrWrapf(ErrInvalid, err, "invalid path for source %q", source.ID)
		}
		if slices.ContainsFunc(enabledPaths, func(existing string) bool {
			return rootsOverlap(existing, rootPath)
		}) {
			return Registry{}, apperr.AppErrMsgf(ErrInvalid, "enabled source root %q overlaps another enabled source root", rootPath)
		}
		enabledPaths = append(enabledPaths, rootPath)

		sourceManifest, err := manifest.Load(rootPath)
		if err != nil {
			return Registry{}, apperr.AppErrWrapf(ErrInvalid, err, "invalid manifest for source %q", source.ID)
		}

		normalized.Sources[i].Path = rootPath
		normalized.Sources[i].Manifest = sourceManifest
	}

	if normalized.Default != "" {
		if _, ok := enabledIDs[normalized.Default]; !ok {
			return Registry{}, apperr.AppErrMsgf(ErrInvalid, "default source %q does not name an enabled source", normalized.Default)
		}
	}

	return normalized, nil
}

func validID(id string) bool {
	return id != "" && !strings.ContainsAny(id, "/:")
}
