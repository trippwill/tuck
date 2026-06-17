package app

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/trippwill/tuck/internal/apperr"
	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pkgref"
	"github.com/trippwill/tuck/internal/plan"
	"github.com/trippwill/tuck/internal/resolve"
	"github.com/trippwill/tuck/internal/state"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{
			name: "manifest missing",
			err:  manifest.ErrMissing,
			code: "manifest_missing",
		},
		{
			name: "manifest invalid",
			err:  manifest.ErrInvalid,
			code: "manifest_invalid",
		},
		{
			name: "state invalid wraps manifest",
			err:  apperr.AppErrWrapf(state.ErrInvalid, manifest.ErrMissing, "invalid registry"),
			code: "state_invalid",
		},
		{
			name: "source root",
			err:  state.ErrSourceRoot,
			code: "source_root_missing",
		},
		{
			name: "state write",
			err:  state.ErrWrite,
			code: "io_error",
		},
		{
			name: "no source",
			err:  resolve.ErrNoSource,
			code: "no_source",
		},
		{
			name: "unknown source",
			err:  resolve.ErrUnknownSource,
			code: "unknown_source",
		},
		{
			name: "fallback",
			err:  errors.New("boom"),
			code: "io_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyError(tt.err)
			if got.Code != tt.code {
				t.Fatalf("classifyError(%v).Code = %q, want %q", tt.err, got.Code, tt.code)
			}
			if got.Message == "" {
				t.Fatalf("classifyError(%v).Message is empty", tt.err)
			}
			if got.Hint == "" {
				t.Fatalf("classifyError(%v).Hint is empty", tt.err)
			}
		})
	}
}

func TestClassifyErrorPreservesDetailMessages(t *testing.T) {
	_, invalidRefErr := pkgref.Parse("bad/ref")
	tests := []struct {
		name       string
		err        error
		code       string
		contains   []string
		notContain string
	}{
		{
			name: "plan apply context",
			err: apperr.AppErrWrapf(
				plan.ErrApply,
				errors.New("permission denied"),
				"could not create symlink %q",
				"/tmp/link",
			),
			code:       "io_error",
			contains:   []string{"could not create symlink", "/tmp/link", "permission denied"},
			notContain: "could not apply target-tree plan",
		},
		{
			name: "state invalid context",
			err:  apperr.AppErrWrapf(state.ErrInvalid, manifest.ErrMissing, "invalid registry"),
			code: "state_invalid",
			contains: []string{
				"invalid registry",
				"missing manifest",
			},
			notContain: "machine source state is invalid",
		},
		{
			name: "manifest missing context",
			err: apperr.AppErrWrapf(
				manifest.ErrMissing,
				errors.New("open /repo/"+manifest.ManifestFilename+": no such file or directory"),
				"could not read manifest %q",
				"/repo/"+manifest.ManifestFilename,
			),
			code:       "manifest_missing",
			contains:   []string{"could not read manifest", "/repo/" + manifest.ManifestFilename, "no such file or directory"},
			notContain: "source manifest is missing",
		},
		{
			name:       "invalid ref context",
			err:        invalidRefErr,
			code:       "invalid_ref",
			contains:   []string{"invalid package ref", "bad/ref"},
			notContain: "package reference is invalid",
		},
		{
			name:       "package not found context",
			err:        packages.AppErrMsgf(packages.ErrPackageNotFound, "package %q not found in source %q", "ssh", "public"),
			code:       "package_not_found",
			contains:   []string{`package "ssh" not found`, `source "public"`},
			notContain: "package not found: package not found",
		},
		{
			name:       "fallback context",
			err:        errors.New("boom"),
			code:       "io_error",
			contains:   []string{"boom"},
			notContain: "runtime error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyError(tt.err)
			if got.Code != tt.code {
				t.Fatalf("classifyError(%v).Code = %q, want %q", tt.err, got.Code, tt.code)
			}
			for _, want := range tt.contains {
				if !strings.Contains(got.Message, want) {
					t.Fatalf("classifyError(%v).Message = %q, want to contain %q", tt.err, got.Message, want)
				}
			}
			if tt.notContain != "" && strings.Contains(got.Message, tt.notContain) {
				t.Fatalf("classifyError(%v).Message = %q, should not contain %q", tt.err, got.Message, tt.notContain)
			}
		})
	}
}

func TestRunMetaJSONUsesSharedEnvelope(t *testing.T) {
	var out bytes.Buffer
	handled, err := runMetaJSON([]string{"tuck", "--version", "--json"}, &out)
	if err != nil {
		t.Fatalf("runMetaJSON() error = %v, want nil", err)
	}
	if !handled {
		t.Fatalf("runMetaJSON() handled = false, want true")
	}

	want := `{"schemaVersion":1,"command":"version","kind":"version","data":{"version":"dev"},"exitCode":0}` + "\n"
	if got := out.String(); got != want {
		t.Fatalf("runMetaJSON() output = %q, want %q", got, want)
	}
}

func TestBuildSourcesData(t *testing.T) {
	got := buildSourcesData(state.Registry{
		Default: "public",
		Sources: []state.Source{
			{
				ID:       "public",
				Path:     "/repo",
				Enabled:  true,
				Manifest: manifest.Manifest{Description: "public dotfiles"},
			},
			{
				ID:      "disabled",
				Path:    "/missing",
				Enabled: false,
			},
		},
	})

	if len(got.Sources) != 2 {
		t.Fatalf("sources length = %d, want 2", len(got.Sources))
	}
	if !got.Sources[0].Default || got.Sources[0].Description != "public dotfiles" {
		t.Fatalf("first source = %#v, want default with description", got.Sources[0])
	}
	if got.Sources[1].Default || got.Sources[1].Enabled || got.Sources[1].Description != "" {
		t.Fatalf("second source = %#v, want disabled non-default with empty description", got.Sources[1])
	}
}
