package manifest

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			if tt.contents != nil {
				if err := os.WriteFile(filepath.Join(repoRoot, "tuck.toml"), []byte(*tt.contents), 0o644); err != nil {
					t.Fatalf("write tuck.toml: %v", err)
				}
			}

			_, err := Load(repoRoot)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Load() error = %v, want errors.Is(..., %v)", err, tt.wantErr)
			}
			var manifestErr *Error
			if !errors.As(err, &manifestErr) {
				t.Fatalf("Load() error = %T, want errors.As(..., *manifest.Error)", err)
			}
			if got := manifestErr.Sentinel(); got != tt.wantErr {
				t.Fatalf("Load() error sentinel = %v, want %v", got, tt.wantErr)
			}
			asTypeErr, ok := errors.AsType[*Error](err)
			if !ok {
				t.Fatalf("Load() error = %T, want errors.AsType[*manifest.Error](...) ok", err)
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

func writeManifest(t *testing.T, contents string) string {
	t.Helper()

	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "tuck.toml"), []byte(contents), 0o644); err != nil {
		t.Fatalf("write tuck.toml: %v", err)
	}
	return repoRoot
}
