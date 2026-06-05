package manifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type ErrManifest string

const (
	ErrInvalid ErrManifest = "invalid manifest"
	ErrMissing ErrManifest = "missing manifest"
)

func (e ErrManifest) Error() string {
	return string(e)
}

func errInvalid(e error) error {
	return errors.Join(ErrInvalid, e)
}

func errMissing(e error) error {
	return errors.Join(ErrMissing, e)
}

type Manifest struct {
	Name        string
	Description string
}

func Load(repoRoot string) (Manifest, error) {
	manifestPath := filepath.Join(repoRoot, "tuck.toml")
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		return Manifest{}, errMissing(err)
	}

	manifest := struct {
		Name        string `toml:"name"`
		Description string `toml:"description"`
	}{}

	if err := toml.Unmarshal(contents, &manifest); err != nil {
		return Manifest{}, errInvalid(err)
	}

	if strings.TrimSpace(manifest.Name) == "" || strings.ContainsAny(manifest.Name, "/:") {
		return Manifest{}, errInvalid(fmt.Errorf("invalid manifest name %q", manifest.Name))
	}

	return Manifest{
		Name:        manifest.Name,
		Description: manifest.Description,
	}, nil
}
