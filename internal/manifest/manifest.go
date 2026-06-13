package manifest

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

//go:generate go run ../../cmd/errgen -type ErrManifest
type ErrManifest string

const (
	ErrInvalid ErrManifest = "invalid manifest"
	ErrExists  ErrManifest = "manifest exists"
	ErrMissing ErrManifest = "missing manifest"
)

type Manifest struct {
	Name        string
	Description string
}

type InitOptions struct {
	Name        string
	Description string
}

type Initialized struct {
	Root     string
	Path     string
	Manifest Manifest
}

func Init(repoRoot string, options InitOptions) (Initialized, error) {
	root, err := initRoot(repoRoot)
	if err != nil {
		return Initialized{}, err
	}
	manifestPath := filepath.Join(root, "tuck.toml")
	name := options.Name
	if name == "" {
		name = filepath.Base(root)
	}
	sourceManifest := Manifest{Name: name, Description: options.Description}
	if err := validate(sourceManifest, manifestPath); err != nil {
		return Initialized{}, err
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return Initialized{}, AppErrWrapf(ErrInvalid, err, "could not create source directory %q", root)
	}
	if _, err := os.Lstat(manifestPath); err == nil {
		return Initialized{}, AppErrMsgf(ErrExists, "manifest already exists at %q", manifestPath)
	} else if !os.IsNotExist(err) {
		return Initialized{}, AppErrWrapf(ErrInvalid, err, "could not inspect manifest %q", manifestPath)
	}

	if err := writeNewManifest(manifestPath, marshal(sourceManifest)); err != nil {
		return Initialized{}, err
	}
	loaded, err := Load(root)
	if err != nil {
		return Initialized{}, err
	}
	return Initialized{Root: root, Path: manifestPath, Manifest: loaded}, nil
}

func Load(repoRoot string) (Manifest, error) {
	manifestPath := filepath.Join(repoRoot, "tuck.toml")
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		return Manifest{}, AppErrWrapf(ErrMissing, err, "could not read manifest %q", manifestPath)
	}

	manifest := struct {
		Name        string `toml:"name"`
		Description string `toml:"description"`
	}{}

	if err := toml.Unmarshal(contents, &manifest); err != nil {
		return Manifest{}, AppErrWrapf(ErrInvalid, err, "could not parse manifest %q", manifestPath)
	}

	loaded := Manifest{
		Name:        manifest.Name,
		Description: manifest.Description,
	}
	if err := validate(loaded, manifestPath); err != nil {
		return Manifest{}, err
	}
	return loaded, nil
}

func initRoot(repoRoot string) (string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return "", AppErrMsg(ErrInvalid, "source path is empty")
	}
	cleanPath := filepath.Clean(repoRoot)
	if cleanPath == "~" || strings.HasPrefix(cleanPath, "~"+string(os.PathSeparator)) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", AppErrWrap(ErrInvalid, err)
		}
		cleanPath = filepath.Join(home, strings.TrimPrefix(cleanPath, "~"))
	}
	if !filepath.IsAbs(cleanPath) {
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return "", AppErrWrap(ErrInvalid, err)
		}
		cleanPath = absPath
	}
	return filepath.Clean(cleanPath), nil
}

func validate(manifest Manifest, manifestPath string) error {
	if strings.TrimSpace(manifest.Name) == "" || strings.ContainsAny(manifest.Name, "/:") {
		return AppErrMsgf(ErrInvalid, "invalid manifest name %q in %q: must be non-empty and cannot contain '/' or ':'", manifest.Name, manifestPath)
	}
	return nil
}

func marshal(manifest Manifest) []byte {
	var b strings.Builder
	b.WriteString("name = ")
	b.WriteString(strconv.Quote(manifest.Name))
	b.WriteString("\n")
	if manifest.Description != "" {
		b.WriteString("description = ")
		b.WriteString(strconv.Quote(manifest.Description))
		b.WriteString("\n")
	}
	return []byte(b.String())
}

func writeNewManifest(manifestPath string, contents []byte) error {
	dir := filepath.Dir(manifestPath)
	tempFile, err := os.CreateTemp(dir, ".tuck.toml.*")
	if err != nil {
		return AppErrWrapf(ErrInvalid, err, "could not create temporary manifest in %q", dir)
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
		return AppErrWrapf(ErrInvalid, err, "could not write temporary manifest %q", tempPath)
	}
	if err := tempFile.Chmod(0o644); err != nil {
		_ = tempFile.Close()
		return AppErrWrapf(ErrInvalid, err, "could not set manifest permissions %q", tempPath)
	}
	if err := tempFile.Close(); err != nil {
		return AppErrWrapf(ErrInvalid, err, "could not close temporary manifest %q", tempPath)
	}
	if err := os.Link(tempPath, manifestPath); err != nil {
		if os.IsExist(err) {
			return AppErrMsgf(ErrExists, "manifest already exists at %q", manifestPath)
		}
		return AppErrWrapf(ErrInvalid, err, "could not create manifest %q", manifestPath)
	}
	removeTemp = false
	if err := os.Remove(tempPath); err != nil {
		return AppErrWrapf(ErrInvalid, err, "could not remove temporary manifest %q", tempPath)
	}
	return nil
}
