package app

import (
	"errors"
	"testing"

	"github.com/trippwill/tuck/internal/apperr"
	"github.com/trippwill/tuck/internal/manifest"
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
			err:  apperr.Wrapf(state.ErrInvalid, manifest.ErrMissing, "invalid registry"),
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
