package packages

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/trippwill/tuck/internal/apperr"
	"github.com/trippwill/tuck/internal/manifest"
)

type Deploy string

const (
	DeploySymlink Deploy = "symlink"
	DeployCopy    Deploy = "copy"
)

const (
	ErrConfigInvalid ErrPackage = "package manifest invalid"
	ErrConfigWrite   ErrPackage = "package manifest write"
)

type FileConfig struct {
	Path   string `json:"path"`
	Deploy Deploy `json:"deploy"`
	Mode   string `json:"mode,omitempty"`
}

type PackageConfig struct {
	ManifestPath string       `json:"manifestPath,omitempty"`
	Files        []FileConfig `json:"files"`
}

func LoadConfig(packageRoot string, entries []Entry) (PackageConfig, error) {
	manifestPath := filepath.Join(packageRoot, manifest.ManifestFilename)
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return PackageConfig{}, nil
		}
		return PackageConfig{}, apperr.AppErrWrapf(ErrConfigInvalid, err, "could not read package manifest %q", manifestPath)
	}

	file := struct {
		Files []struct {
			Path   string `toml:"path"`
			Deploy string `toml:"deploy"`
			Mode   string `toml:"mode"`
		} `toml:"file"`
	}{}
	if err := toml.Unmarshal(contents, &file); err != nil {
		return PackageConfig{}, apperr.AppErrWrapf(ErrConfigInvalid, err, "could not parse package manifest %q", manifestPath)
	}

	leaf := leafSet(entries)
	seen := make(map[string]struct{}, len(file.Files))
	config := PackageConfig{ManifestPath: manifestPath}
	for _, entry := range file.Files {
		rel, err := cleanConfigPath(entry.Path)
		if err != nil {
			return PackageConfig{}, apperr.AppErrMsgf(ErrConfigInvalid, "invalid package manifest %q: invalid file path %q", manifestPath, entry.Path)
		}
		if _, ok := seen[rel]; ok {
			return PackageConfig{}, apperr.AppErrMsgf(ErrConfigInvalid, "invalid package manifest %q: duplicate file path %q", manifestPath, rel)
		}
		seen[rel] = struct{}{}
		if _, ok := leaf[rel]; !ok {
			return PackageConfig{}, apperr.AppErrMsgf(ErrConfigInvalid, "invalid package manifest %q: file path %q is not a package leaf", manifestPath, rel)
		}
		deploy := DeploySymlink
		if entry.Deploy != "" {
			deploy = Deploy(entry.Deploy)
		}
		if deploy != DeploySymlink && deploy != DeployCopy {
			return PackageConfig{}, apperr.AppErrMsgf(ErrConfigInvalid, "invalid package manifest %q: unknown deploy %q", manifestPath, entry.Deploy)
		}
		mode, err := NormalizeMode(entry.Mode)
		if err != nil {
			return PackageConfig{}, apperr.AppErrMsgf(ErrConfigInvalid, "invalid package manifest %q: invalid mode %q", manifestPath, entry.Mode)
		}
		config.Files = append(config.Files, FileConfig{Path: rel, Deploy: deploy, Mode: mode})
	}
	sortConfig(config.Files)
	return config, nil
}

func ApplyConfig(entries []Entry, config PackageConfig) []Entry {
	byPath := make(map[string]FileConfig, len(config.Files))
	for _, file := range config.Files {
		byPath[file.Path] = file
	}
	configured := make([]Entry, len(entries))
	for i, entry := range entries {
		configured[i] = entry
		if entry.Dir {
			continue
		}
		configured[i].Deploy = DeploySymlink
		if file, ok := byPath[entry.Rel]; ok {
			configured[i].Deploy = file.Deploy
			configured[i].Mode = file.Mode
		}
	}
	return configured
}

func SaveConfig(packageRoot string, config PackageConfig) error {
	sortConfig(config.Files)
	manifestPath := filepath.Join(packageRoot, manifest.ManifestFilename)
	if len(config.Files) == 0 {
		if err := os.Remove(manifestPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return apperr.AppErrWrapf(ErrConfigWrite, err, "could not remove package manifest %q", manifestPath)
		}
		return nil
	}
	if err := os.MkdirAll(packageRoot, 0o755); err != nil {
		return apperr.AppErrWrapf(ErrConfigWrite, err, "could not create package directory %q", packageRoot)
	}
	contents := marshalConfig(config)
	tempFile, err := os.CreateTemp(packageRoot, manifest.ManifestFilename+".*")
	if err != nil {
		return apperr.AppErrWrapf(ErrConfigWrite, err, "could not create temporary package manifest in %q", packageRoot)
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
		return apperr.AppErrWrapf(ErrConfigWrite, err, "could not write temporary package manifest %q", tempPath)
	}
	if err := tempFile.Chmod(0o644); err != nil {
		_ = tempFile.Close()
		return apperr.AppErrWrapf(ErrConfigWrite, err, "could not set package manifest permissions %q", tempPath)
	}
	if err := tempFile.Close(); err != nil {
		return apperr.AppErrWrapf(ErrConfigWrite, err, "could not close temporary package manifest %q", tempPath)
	}
	if err := os.Rename(tempPath, manifestPath); err != nil {
		return apperr.AppErrWrapf(ErrConfigWrite, err, "could not replace package manifest %q", manifestPath)
	}
	removeTemp = false
	return nil
}

func NormalizeMode(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	value, err := strconv.ParseUint(raw, 8, 32)
	if err != nil || value > 0o777 {
		return "", fmt.Errorf("invalid mode")
	}
	return fmt.Sprintf("%04o", value), nil
}

func NormalizeModeFlag(raw string, base string) (string, error) {
	if mode, err := NormalizeMode(raw); err == nil {
		return mode, nil
	}
	value, err := strconv.ParseUint(base, 8, 32)
	if err != nil || value > 0o777 {
		return "", fmt.Errorf("invalid mode")
	}
	for _, clause := range strings.Split(raw, ",") {
		next, err := applyModeClause(os.FileMode(value), strings.TrimSpace(clause))
		if err != nil {
			return "", err
		}
		value = uint64(next)
	}
	return fmt.Sprintf("%04o", value), nil
}

func applyModeClause(base os.FileMode, clause string) (os.FileMode, error) {
	if clause == "" {
		return 0, fmt.Errorf("invalid mode")
	}
	i := 0
	who := os.FileMode(0)
	for ; i < len(clause); i++ {
		switch clause[i] {
		case 'u':
			who |= 0o700
		case 'g':
			who |= 0o070
		case 'o':
			who |= 0o007
		case 'a':
			who |= 0o777
		default:
			goto op
		}
	}
op:
	if who == 0 {
		who = 0o777
	}
	if i >= len(clause) {
		return 0, fmt.Errorf("invalid mode")
	}
	op := clause[i]
	if op != '+' && op != '-' && op != '=' {
		return 0, fmt.Errorf("invalid mode")
	}
	i++
	perms := os.FileMode(0)
	for ; i < len(clause); i++ {
		switch clause[i] {
		case 'r':
			perms |= 0o444
		case 'w':
			perms |= 0o222
		case 'x':
			perms |= 0o111
		default:
			return 0, fmt.Errorf("invalid mode")
		}
	}
	perms &= who
	switch op {
	case '+':
		base |= perms
	case '-':
		base &^= perms
	case '=':
		base = (base &^ who) | perms
	}
	return base.Perm(), nil
}

func ModeFromFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04o", info.Mode().Perm()), nil
}

func ConfiguredFile(config PackageConfig, rel string) (FileConfig, bool) {
	rel, err := cleanConfigPath(rel)
	if err != nil {
		return FileConfig{}, false
	}
	for _, file := range config.Files {
		if file.Path == rel {
			return file, true
		}
	}
	return FileConfig{}, false
}

func SetFileConfig(config PackageConfig, file FileConfig) PackageConfig {
	for i, existing := range config.Files {
		if existing.Path == file.Path {
			config.Files[i] = file
			sortConfig(config.Files)
			return config
		}
	}
	config.Files = append(config.Files, file)
	sortConfig(config.Files)
	return config
}

func RemoveFileConfig(config PackageConfig, rel string) PackageConfig {
	files := config.Files[:0]
	for _, file := range config.Files {
		if file.Path != rel {
			files = append(files, file)
		}
	}
	config.Files = files
	return config
}

func cleanConfigPath(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" || filepath.IsAbs(raw) {
		return "", fmt.Errorf("invalid path")
	}
	rel := filepath.Clean(raw)
	if rel == "." || rel == manifest.ManifestFilename || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", fmt.Errorf("invalid path")
	}
	return rel, nil
}

func leafSet(entries []Entry) map[string]struct{} {
	leaf := make(map[string]struct{})
	for _, entry := range entries {
		if !entry.Dir {
			leaf[entry.Rel] = struct{}{}
		}
	}
	return leaf
}

func sortConfig(files []FileConfig) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
}

func marshalConfig(config PackageConfig) []byte {
	var b bytes.Buffer
	for i, file := range config.Files {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("[[file]]\n")
		b.WriteString("path = ")
		b.WriteString(strconv.Quote(file.Path))
		b.WriteString("\n")
		b.WriteString("deploy = ")
		b.WriteString(strconv.Quote(string(file.Deploy)))
		b.WriteString("\n")
		if file.Mode != "" {
			b.WriteString("mode = ")
			b.WriteString(strconv.Quote(file.Mode))
			b.WriteString("\n")
		}
	}
	return b.Bytes()
}
