package sourcecmd

import (
	"strings"
	"testing"

	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/state"
)

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

func TestRenderListStylesHeadersAndBooleans(t *testing.T) {
	got := renderList(output.NewConsole(output.Invocation{Command: "source list"}, true), ListPayload{
		Registry: state.Registry{
			Default: "public",
			Sources: []state.Source{{
				ID:      "public",
				Path:    "/repo",
				Enabled: true,
			}},
		},
	})

	if !strings.Contains(got, "\x1b[1mID      \x1b[0m \x1b[1mDEFAULT \x1b[0m \x1b[1mENABLED \x1b[0m \x1b[1mPATH\x1b[0m") {
		t.Fatalf("renderList() = %q, want styled header", got)
	}
	if !strings.Contains(got, "\x1b[35mpublic  \x1b[0m \x1b[32myes     \x1b[0m \x1b[32myes     \x1b[0m \x1b[36m/repo\x1b[0m") {
		t.Fatalf("renderList() = %q, want styled source row", got)
	}
}
