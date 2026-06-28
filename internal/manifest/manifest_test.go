package manifest

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/trippwill/tuck/internal/apperr"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		want     Manifest
	}{
		{
			name:     "minimal manifest",
			contents: "name = \"public\"\n",
			want:     Manifest{Name: "public"},
		},
		{
			name: "optional description",
			contents: `name = "public"
description = "public dotfiles"
`,
			want: Manifest{Name: "public", Description: "public dotfiles"},
		},
		{
			name: "unknown keys are ignored",
			contents: `name = "public"
description = "public dotfiles"
future = "ignored"

[security]
policy = "reserved"
`,
			want: Manifest{Name: "public", Description: "public dotfiles"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot := writeManifest(t, tt.contents)

			got, err := Load(repoRoot)
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}
			if got != tt.want {
				t.Fatalf("Load() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name     string
		contents *string
		wantErr  ErrManifest
	}{
		{
			name:    "missing manifest",
			wantErr: ErrMissing,
		},
		{
			name:     "malformed toml",
			contents: new("name = \"public\n"),
			wantErr:  ErrInvalid,
		},
		{
			name:     "missing name",
			contents: new("description = \"public dotfiles\"\n"),
			wantErr:  ErrInvalid,
		},
		{
			name:     "empty name",
			contents: new("name = \"\"\n"),
			wantErr:  ErrInvalid,
		},
		{
			name:     "name with path separator",
			contents: new("name = \"pub/lic\"\n"),
			wantErr:  ErrInvalid,
		},
		{
			name:     "name with colon",
			contents: new("name = \"public:home\"\n"),
			wantErr:  ErrInvalid,
		},
		{
			name:     "reserved manifest filename",
			contents: new("name = \".tuck.toml\"\n"),
			wantErr:  ErrInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			if tt.contents != nil {
				if err := os.WriteFile(filepath.Join(repoRoot, ManifestFilename), []byte(*tt.contents), 0o644); err != nil {
					t.Fatalf("write %s: %v", ManifestFilename, err)
				}
			}

			_, err := Load(repoRoot)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Load() error = %v, want errors.Is(..., %v)", err, tt.wantErr)
			}
			var manifestErr *apperr.Error[ErrManifest]
			if !errors.As(err, &manifestErr) {
				t.Fatalf("Load() error = %T, want errors.As(..., *apperr.Error[manifest.ErrManifest])", err)
			}
			if got := manifestErr.Sentinel(); got != tt.wantErr {
				t.Fatalf("Load() error sentinel = %v, want %v", got, tt.wantErr)
			}
			asTypeErr, ok := errors.AsType[*apperr.Error[ErrManifest]](err)
			if !ok {
				t.Fatalf("Load() error = %T, want errors.AsType[*apperr.Error[manifest.ErrManifest]](...) ok", err)
			}
			if got := asTypeErr.Sentinel(); got != tt.wantErr {
				t.Fatalf("Load() AsType error sentinel = %v, want %v", got, tt.wantErr)
			}
			if tt.contents == nil && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("Load() missing error = %v, want errors.Is(..., os.ErrNotExist)", err)
			}
		})
	}
}

func TestInitCreatesManifestWithDefaultName(t *testing.T) {
	repoRoot := filepath.Join(t.TempDir(), "dotfiles")

	got, err := Init(repoRoot, InitOptions{})
	if err != nil {
		t.Fatalf("Init() error = %v, want nil", err)
	}
	if got.Root != repoRoot {
		t.Fatalf("Init() root = %q, want %q", got.Root, repoRoot)
	}
	if got.Path != filepath.Join(repoRoot, ManifestFilename) {
		t.Fatalf("Init() path = %q, want %s under repo root", got.Path, ManifestFilename)
	}
	if got.Manifest != (Manifest{Name: "dotfiles"}) {
		t.Fatalf("Init() manifest = %#v, want default basename name", got.Manifest)
	}
	loaded, err := Load(repoRoot)
	if err != nil {
		t.Fatalf("Load() after Init() error = %v", err)
	}
	if loaded != got.Manifest {
		t.Fatalf("Load() after Init() = %#v, want %#v", loaded, got.Manifest)
	}
}

func TestInitWritesExplicitNameAndDescriptionDeterministically(t *testing.T) {
	repoRoot := filepath.Join(t.TempDir(), "repo")

	got, err := Init(repoRoot, InitOptions{Name: "public", Description: "public dotfiles"})
	if err != nil {
		t.Fatalf("Init() error = %v, want nil", err)
	}
	if got.Manifest != (Manifest{Name: "public", Description: "public dotfiles"}) {
		t.Fatalf("Init() manifest = %#v, want explicit fields", got.Manifest)
	}
	contents, err := os.ReadFile(filepath.Join(repoRoot, ManifestFilename))
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", ManifestFilename, err)
	}
	want := "name = \"public\"\ndescription = \"public dotfiles\"\n"
	if string(contents) != want {
		t.Fatalf("%s = %q, want %q", ManifestFilename, contents, want)
	}
}

func TestInitRejectsInvalidNameAndExistingManifest(t *testing.T) {
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if _, err := Init(repoRoot, InitOptions{Name: "bad/name"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Init() invalid name error = %v, want ErrInvalid", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, ManifestFilename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Init() invalid name wrote manifest, stat err = %v", err)
	}

	repoRoot = writeManifest(t, "name = \"public\"\n")
	before, err := os.ReadFile(filepath.Join(repoRoot, ManifestFilename))
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", ManifestFilename, err)
	}
	if _, err := Init(repoRoot, InitOptions{Name: "other"}); !errors.Is(err, ErrExists) {
		t.Fatalf("Init() existing manifest error = %v, want ErrExists", err)
	}
	after, err := os.ReadFile(filepath.Join(repoRoot, ManifestFilename))
	if err != nil {
		t.Fatalf("ReadFile(%s) after existing error = %v", ManifestFilename, err)
	}
	if string(after) != string(before) {
		t.Fatalf("existing %s changed:\nbefore: %s\nafter: %s", ManifestFilename, before, after)
	}
}

func TestGenericAppErrorSupportsTypedInspection(t *testing.T) {
	err := apperr.AppErrMsgf(ErrInvalid, "invalid manifest name %q", "bad/name")

	var manifestErr *apperr.Error[ErrManifest]
	if !errors.As(err, &manifestErr) {
		t.Fatalf("errors.As(err, *apperr.Error[manifest.ErrManifest]) = false, want true")
	}
	if got := manifestErr.Sentinel(); got != ErrInvalid {
		t.Fatalf("Sentinel() = %v, want %v", got, ErrInvalid)
	}
}

func writeManifest(t *testing.T, contents string) string {
	t.Helper()

	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, ManifestFilename), []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", ManifestFilename, err)
	}
	return repoRoot
}
