package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/trippwill/tuck/internal/manifest"
)

func TestLoadAbsentState(t *testing.T) {
	withStateHome(t)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if len(got.Sources) != 0 {
		t.Fatalf("Load() sources = %#v, want empty registry", got.Sources)
	}
	if got.Default != "" {
		t.Fatalf("Load() default = %q, want empty default", got.Default)
	}
}

func TestLoadSources(t *testing.T) {
	stateRoot := withStateHome(t)
	omittedEnabledRepo := writeSourceRepo(t, "omitted", "omitted description")
	explicitEnabledRepo := writeSourceRepo(t, "explicit", "explicit description")
	disabledRepo := writeSourceRepo(t, "disabled", "disabled description")
	writeSources(t, stateRoot, stateFile("omitted",
		sourceBlock("omitted", omittedEnabledRepo, nil)+
			sourceBlock("explicit", explicitEnabledRepo, new(true))+
			sourceBlock("disabled", disabledRepo, new(false)),
	),
	)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if len(got.Sources) != 3 {
		t.Fatalf("Load() sources = %#v, want 3 entries", got.Sources)
	}
	if got.Default != "omitted" {
		t.Fatalf("Load() default = %q, want %q", got.Default, "omitted")
	}

	omitted := requireSource(t, got, "omitted")
	if !omitted.Enabled {
		t.Fatalf("source with omitted enabled flag was disabled, want enabled")
	}
	if omitted.Path != canonical(t, omittedEnabledRepo) {
		t.Fatalf("omitted source path = %q, want canonical %q", omitted.Path, canonical(t, omittedEnabledRepo))
	}
	if omitted.Manifest != (manifest.Manifest{Name: "omitted", Description: "omitted description"}) {
		t.Fatalf("omitted manifest = %#v, want parsed manifest", omitted.Manifest)
	}

	explicit := requireSource(t, got, "explicit")
	if !explicit.Enabled {
		t.Fatalf("source with enabled = true was disabled, want enabled")
	}

	disabled := requireSource(t, got, "disabled")
	if disabled.Enabled {
		t.Fatalf("source with enabled = false was enabled")
	}
}

func TestLoadAllowsSiblingPrefixPaths(t *testing.T) {
	stateRoot := withStateHome(t)
	base := t.TempDir()
	repo := writeSourceRepoAt(t, filepath.Join(base, "repo"), "repo", "")
	repoPrivate := writeSourceRepoAt(t, filepath.Join(base, "repo-private"), "repo-private", "")
	writeSources(t, stateRoot,
		sourceBlock("repo", repo, nil)+
			sourceBlock("repo-private", repoPrivate, nil),
	)

	if _, err := Load(); err != nil {
		t.Fatalf("Load() error = %v, want nil for sibling prefix paths", err)
	}
}

func TestLoadValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		sources func(t *testing.T) string
	}{
		{
			name: "malformed state",
			sources: func(t *testing.T) string {
				return "[[source]\n"
			},
		},
		{
			name: "missing enabled source id",
			sources: func(t *testing.T) string {
				repo := writeSourceRepo(t, "public", "")
				return `[[source]]
path = ` + strconv.Quote(repo) + "\n"
			},
		},
		{
			name: "empty enabled source id",
			sources: func(t *testing.T) string {
				repo := writeSourceRepo(t, "public", "")
				return sourceBlock("", repo, nil)
			},
		},
		{
			name: "enabled source id with path separator",
			sources: func(t *testing.T) string {
				repo := writeSourceRepo(t, "public", "")
				return sourceBlock("pub/lic", repo, nil)
			},
		},
		{
			name: "enabled source id with colon",
			sources: func(t *testing.T) string {
				repo := writeSourceRepo(t, "public", "")
				return sourceBlock("public:home", repo, nil)
			},
		},
		{
			name: "duplicate enabled source ids",
			sources: func(t *testing.T) string {
				first := writeSourceRepo(t, "public", "")
				second := writeSourceRepo(t, "public", "")
				return sourceBlock("public", first, nil) +
					sourceBlock("public", second, nil)
			},
		},
		{
			name: "default names missing source",
			sources: func(t *testing.T) string {
				repo := writeSourceRepo(t, "public", "")
				return stateFile("missing", sourceBlock("public", repo, nil))
			},
		},
		{
			name: "default names disabled source",
			sources: func(t *testing.T) string {
				repo := writeSourceRepo(t, "disabled", "")
				return stateFile("disabled", sourceBlock("disabled", repo, new(false)))
			},
		},
		{
			name: "missing enabled source path",
			sources: func(t *testing.T) string {
				return sourceBlock("missing", filepath.Join(t.TempDir(), "missing"), nil)
			},
		},
		{
			name: "missing enabled source manifest",
			sources: func(t *testing.T) string {
				return sourceBlock("public", t.TempDir(), nil)
			},
		},
		{
			name: "invalid enabled source manifest",
			sources: func(t *testing.T) string {
				repo := t.TempDir()
				writeFile(t, filepath.Join(repo, "tuck.toml"), "description = \"missing name\"\n")
				return sourceBlock("public", repo, nil)
			},
		},
		{
			name: "equal enabled source roots overlap",
			sources: func(t *testing.T) string {
				repo := writeSourceRepo(t, "public", "")
				return sourceBlock("public", repo, nil) +
					sourceBlock("other", repo, nil)
			},
		},
		{
			name: "nested enabled source roots overlap",
			sources: func(t *testing.T) string {
				parent := writeSourceRepo(t, "parent", "")
				child := writeSourceRepoAt(t, filepath.Join(parent, "child"), "child", "")
				return sourceBlock("parent", parent, nil) +
					sourceBlock("child", child, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateRoot := withStateHome(t)
			writeSources(t, stateRoot, tt.sources(t))

			_, err := Load()
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("Load() error = %v, want errors.Is(..., %v)", err, ErrInvalid)
			}
		})
	}
}

func TestLoadDisabledEntriesDoNotParticipateInEnabledOnlyValidation(t *testing.T) {
	stateRoot := withStateHome(t)
	enabledRepo := writeSourceRepo(t, "public", "")
	invalidManifestRepo := t.TempDir()
	writeFile(t, filepath.Join(invalidManifestRepo, "tuck.toml"), "description = \"missing name\"\n")
	missingRepo := filepath.Join(t.TempDir(), "missing")
	writeSources(t, stateRoot, stateFile("public",
		sourceBlock("public", enabledRepo, nil)+
			sourceBlock("public", missingRepo, new(false))+
			sourceBlock("invalid", invalidManifestRepo, new(false)),
	),
	)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil for disabled entries with duplicate id, missing path, and invalid manifest", err)
	}
	if len(got.Sources) != 3 {
		t.Fatalf("Load() sources = %#v, want disabled entries preserved", got.Sources)
	}
	if got.Default != "public" {
		t.Fatalf("Load() default = %q, want %q", got.Default, "public")
	}
}

func withStateHome(t *testing.T) string {
	t.Helper()

	stateRoot := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateRoot)
	t.Setenv("TUCK_TEST_STATE_DIR", "")
	return stateRoot
}

func writeSources(t *testing.T, stateRoot, contents string) {
	t.Helper()

	writeFile(t, filepath.Join(stateRoot, "tuck", "sources.toml"), contents)
}

func writeSourceRepo(t *testing.T, name, description string) string {
	t.Helper()

	return writeSourceRepoAt(t, filepath.Join(t.TempDir(), name), name, description)
}

func writeSourceRepoAt(t *testing.T, path, name, description string) string {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir source repo: %v", err)
	}

	var manifest strings.Builder
	manifest.WriteString("name = ")
	manifest.WriteString(strconv.Quote(name))
	manifest.WriteString("\n")

	if description != "" {
		manifest.WriteString("description = ")
		manifest.WriteString(strconv.Quote(description))
		manifest.WriteString("\n")
	}
	writeFile(t, filepath.Join(path, "tuck.toml"), manifest.String())
	return path
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func stateFile(defaultID, sources string) string {
	var b strings.Builder
	if defaultID != "" {
		b.WriteString("default = ")
		b.WriteString(strconv.Quote(defaultID))
		b.WriteString("\n\n")
	}
	b.WriteString(sources)
	return b.String()
}

func sourceBlock(id, path string, enabled *bool) string {
	var b strings.Builder
	b.WriteString("[[source]]\n")
	b.WriteString("id = ")
	b.WriteString(strconv.Quote(id))
	b.WriteString("\n")
	b.WriteString("path = ")
	b.WriteString(strconv.Quote(path))
	b.WriteString("\n")
	if enabled != nil {
		fmt.Fprintf(&b, "enabled = %t\n", *enabled)
	}
	return b.String()
}

func requireSource(t *testing.T, registry Registry, id string) Source {
	t.Helper()

	for _, source := range registry.Sources {
		if source.ID == id {
			return source
		}
	}
	t.Fatalf("source %q not found in %#v", id, registry.Sources)
	return Source{}
}

func canonical(t *testing.T, path string) string {
	t.Helper()

	canonicalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("canonicalize %s: %v", path, err)
	}
	return canonicalPath
}
