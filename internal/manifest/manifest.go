package manifest

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/trippwill/tuck/internal/apperr"
)

type Error = apperr.Error[ErrManifest]
type ErrManifest string

const (
	ErrInvalid ErrManifest = "invalid manifest"
	ErrMissing ErrManifest = "missing manifest"
)

func (e ErrManifest) Error() string { return string(e) }

type Manifest struct {
	Name        string
	Description string
}

func Load(repoRoot string) (Manifest, error) {
	manifestPath := filepath.Join(repoRoot, "tuck.toml")
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		return Manifest{}, apperr.Wrapf(ErrMissing, err, "could not read manifest %q", manifestPath)
	}

	manifest := struct {
		Name        string `toml:"name"`
		Description string `toml:"description"`
	}{}

	if err := toml.Unmarshal(contents, &manifest); err != nil {
		return Manifest{}, apperr.Wrapf(ErrInvalid, err, "could not parse manifest %q", manifestPath)
	}

	if strings.TrimSpace(manifest.Name) == "" || strings.ContainsAny(manifest.Name, "/:") {
		return Manifest{}, apperr.Wrapf(ErrInvalid, nil, "invalid manifest name %q in %q: must be non-empty and cannot contain '/' or ':'", manifest.Name, manifestPath)
	}

	return Manifest{
		Name:        manifest.Name,
		Description: manifest.Description,
	}, nil
}
