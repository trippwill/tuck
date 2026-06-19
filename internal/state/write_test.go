package state

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/trippwill/tuck/internal/manifest"
)

func TestSaveWritesNormalizedState(t *testing.T) {
	stateRoot := withStateHome(t)
	publicRepo := writeSourceRepo(t, "public", "public dotfiles")
	disabledRepo := filepath.Join(t.TempDir(), "disabled")

	err := Save(Registry{
		Default: "public",
		Sources: []Source{
			{ID: "public", Path: publicRepo, Enabled: true},
			{ID: "disabled", Path: disabledRepo, Enabled: false},
		},
	})
	if err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	got := readSourcesFile(t, stateRoot)
	want := `default = "public"

[[source]]
id = "public"
path = ` + quote(canonical(t, publicRepo)) + `
enabled = true

[[source]]
id = "disabled"
path = ` + quote(disabledRepo) + `
enabled = false
`
	if got != want {
		t.Fatalf("sources.toml =\n%s\nwant:\n%s", got, want)
	}
}

func TestSaveDoesNotReplaceExistingStateOnValidationFailure(t *testing.T) {
	stateRoot := withStateHome(t)
	originalRepo := writeSourceRepo(t, "public", "")
	writeSources(t, stateRoot, stateFile("public", sourceBlock("public", originalRepo, nil)))
	before := readSourcesFile(t, stateRoot)

	err := Save(Registry{
		Sources: []Source{
			{ID: "bad/id", Path: originalRepo, Enabled: true},
		},
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("Save() error = %v, want errors.Is(..., %v)", err, ErrInvalid)
	}
	after := readSourcesFile(t, stateRoot)
	if after != before {
		t.Fatalf("sources.toml changed after failed Save():\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestAddSourceAddsDefaultSource(t *testing.T) {
	stateRoot := withStateHome(t)
	repo := writeSourceRepo(t, "public", "public dotfiles")

	registry, source, err := AddSource(repo, true)
	if err != nil {
		t.Fatalf("AddSource() error = %v, want nil", err)
	}
	if source.ID != "public" || source.Path != canonical(t, repo) || !source.Enabled {
		t.Fatalf("AddSource() source = %#v, want enabled public at canonical path", source)
	}
	if registry.Default != "public" {
		t.Fatalf("AddSource() default = %q, want public", registry.Default)
	}

	got := readSourcesFile(t, stateRoot)
	if !strings.Contains(got, "default = \"public\"\n") {
		t.Fatalf("sources.toml = %q, want top-level default", got)
	}
	if !strings.Contains(got, "path = "+quote(canonical(t, repo))+"\n") {
		t.Fatalf("sources.toml = %q, want canonical path", got)
	}
}

func TestAddSourceIsIdempotentForSameIDAndPath(t *testing.T) {
	stateRoot := withStateHome(t)
	repo := writeSourceRepo(t, "public", "")
	if _, _, err := AddSource(repo, true); err != nil {
		t.Fatalf("first AddSource() error = %v, want nil", err)
	}
	before := readSourcesFile(t, stateRoot)

	registry, _, err := AddSource(repo, true)
	if err != nil {
		t.Fatalf("second AddSource() error = %v, want nil", err)
	}
	if len(registry.Sources) != 1 {
		t.Fatalf("AddSource() sources = %#v, want one source", registry.Sources)
	}
	after := readSourcesFile(t, stateRoot)
	if after != before {
		t.Fatalf("idempotent AddSource() changed state:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestAddSourceReenablesDisabledEntryWithSameIDAndPath(t *testing.T) {
	stateRoot := withStateHome(t)
	repo := writeSourceRepo(t, "public", "")
	writeSources(t, stateRoot, sourceBlock("public", repo, new(false)))

	registry, source, err := AddSource(repo, false)
	if err != nil {
		t.Fatalf("AddSource() error = %v, want nil", err)
	}
	if !source.Enabled {
		t.Fatalf("AddSource() source enabled = false, want true")
	}
	got := requireSource(t, registry, "public")
	if !got.Enabled {
		t.Fatalf("registry source enabled = false, want true")
	}
	if !strings.Contains(readSourcesFile(t, stateRoot), "enabled = true\n") {
		t.Fatalf("sources.toml did not re-enable source:\n%s", readSourcesFile(t, stateRoot))
	}
}

func TestAddSourceRejectsSameIDWithDifferentPathWithoutChangingState(t *testing.T) {
	stateRoot := withStateHome(t)
	first := writeSourceRepo(t, "public", "")
	second := writeSourceRepo(t, "public", "other checkout")
	if _, _, err := AddSource(first, true); err != nil {
		t.Fatalf("first AddSource() error = %v, want nil", err)
	}
	before := readSourcesFile(t, stateRoot)

	_, _, err := AddSource(second, true)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("AddSource() error = %v, want errors.Is(..., %v)", err, ErrInvalid)
	}
	after := readSourcesFile(t, stateRoot)
	if after != before {
		t.Fatalf("sources.toml changed after id collision:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestAddSourceLeavesStateUnchangedWhenExistingEntryIsInvalid(t *testing.T) {
	stateRoot := withStateHome(t)
	repo := writeSourceRepo(t, "public", "")
	writeSources(t, stateRoot, sourceBlock("public", filepath.Join(t.TempDir(), "missing"), new(false)))
	before := readSourcesFile(t, stateRoot)

	_, _, err := AddSource(repo, false)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("AddSource() error = %v, want errors.Is(..., %v)", err, ErrInvalid)
	}
	if after := readSourcesFile(t, stateRoot); after != before {
		t.Fatalf("sources.toml changed after failed mutation:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestAddSourceClassifiesSourceRootAndManifestErrors(t *testing.T) {
	withStateHome(t)

	_, _, err := AddSource(filepath.Join(t.TempDir(), "missing"), false)
	if !errors.Is(err, ErrSourceRoot) {
		t.Fatalf("AddSource() missing root error = %v, want errors.Is(..., %v)", err, ErrSourceRoot)
	}

	_, _, err = AddSource(t.TempDir(), false)
	if err == nil {
		t.Fatalf("AddSource() missing manifest error = nil, want error")
	}
}

func TestAddSourceWithInitCreatesManifestAndRegistersSource(t *testing.T) {
	stateRoot := withStateHome(t)
	repo := filepath.Join(t.TempDir(), "dotfiles")

	registry, source, err := AddSourceWithInit(repo, true, manifest.InitOptions{
		Name:        "public",
		Description: "public dotfiles",
	})
	if err != nil {
		t.Fatalf("AddSourceWithInit() error = %v, want nil", err)
	}
	if source.ID != "public" || !source.Enabled || source.Manifest.Description != "public dotfiles" {
		t.Fatalf("AddSourceWithInit() source = %#v, want enabled public with description", source)
	}
	if registry.Default != "public" {
		t.Fatalf("AddSourceWithInit() default = %q, want public", registry.Default)
	}
	loaded, err := manifest.Load(repo)
	if err != nil {
		t.Fatalf("manifest.Load() after AddSourceWithInit() error = %v", err)
	}
	if loaded != (manifest.Manifest{Name: "public", Description: "public dotfiles"}) {
		t.Fatalf("manifest.Load() after AddSourceWithInit() = %#v", loaded)
	}
	got := readSourcesFile(t, stateRoot)
	if !strings.Contains(got, "id = \"public\"") || !strings.Contains(got, "default = \"public\"") {
		t.Fatalf("sources.toml after AddSourceWithInit() =\n%s\nwant registered default public", got)
	}
}

func TestAddSourceWithInitDoesNotHideInvalidExistingManifest(t *testing.T) {
	stateRoot := withStateHome(t)
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, manifest.ManifestFilename), "description = \"missing name\"\n")

	_, _, err := AddSourceWithInit(repo, true, manifest.InitOptions{Name: "public"})
	if !errors.Is(err, manifest.ErrInvalid) {
		t.Fatalf("AddSourceWithInit() error = %v, want manifest.ErrInvalid", err)
	}
	if _, statErr := os.Stat(filepath.Join(stateRoot, "tuck", "sources.toml")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("sources.toml exists after invalid manifest, stat err = %v", statErr)
	}
}

func TestRemoveSourceRemovesEntryAndClearsDefault(t *testing.T) {
	stateRoot := withStateHome(t)
	publicRepo := writeSourceRepo(t, "public", "")
	privateRepo := writeSourceRepo(t, "private", "")
	writeSources(t, stateRoot, stateFile("public",
		sourceBlock("public", publicRepo, nil)+
			sourceBlock("private", privateRepo, nil)+
			sourceBlock("disabled", filepath.Join(t.TempDir(), "disabled"), new(false)),
	))

	registry, removed, ok, err := RemoveSource("public")
	if err != nil {
		t.Fatalf("RemoveSource() error = %v, want nil", err)
	}
	if !ok {
		t.Fatal("RemoveSource() ok = false, want true")
	}
	if removed.ID != "public" {
		t.Fatalf("RemoveSource() removed = %#v, want public", removed)
	}
	if registry.Default != "" {
		t.Fatalf("RemoveSource() default = %q, want cleared", registry.Default)
	}
	if len(registry.Sources) != 2 {
		t.Fatalf("RemoveSource() sources = %#v, want private and disabled", registry.Sources)
	}
	got := readSourcesFile(t, stateRoot)
	if strings.Contains(got, "id = \"public\"") || strings.Contains(got, "default =") {
		t.Fatalf("sources.toml after RemoveSource(public) =\n%s\nwant public/default removed", got)
	}
	if !strings.Contains(got, "id = \"private\"") || !strings.Contains(got, "id = \"disabled\"") {
		t.Fatalf("sources.toml after RemoveSource(public) =\n%s\nwant remaining sources preserved", got)
	}
}

func TestRemoveSourcePreservesOtherDefault(t *testing.T) {
	stateRoot := withStateHome(t)
	publicRepo := writeSourceRepo(t, "public", "")
	privateRepo := writeSourceRepo(t, "private", "")
	writeSources(t, stateRoot, stateFile("private",
		sourceBlock("public", publicRepo, nil)+
			sourceBlock("private", privateRepo, nil),
	))

	registry, _, ok, err := RemoveSource("public")
	if err != nil {
		t.Fatalf("RemoveSource() error = %v, want nil", err)
	}
	if !ok {
		t.Fatal("RemoveSource() ok = false, want true")
	}
	if registry.Default != "private" {
		t.Fatalf("RemoveSource() default = %q, want private", registry.Default)
	}
	if strings.Contains(readSourcesFile(t, stateRoot), "id = \"public\"") {
		t.Fatalf("sources.toml after RemoveSource(public) still contains public:\n%s", readSourcesFile(t, stateRoot))
	}
}

func TestRemoveSourceUnknownLeavesStateUnchanged(t *testing.T) {
	stateRoot := withStateHome(t)
	publicRepo := writeSourceRepo(t, "public", "")
	writeSources(t, stateRoot, stateFile("public", sourceBlock("public", publicRepo, nil)))
	before := readSourcesFile(t, stateRoot)

	_, _, ok, err := RemoveSource("missing")
	if err != nil {
		t.Fatalf("RemoveSource() error = %v, want nil unknown result", err)
	}
	if ok {
		t.Fatal("RemoveSource() ok = true, want false")
	}
	if after := readSourcesFile(t, stateRoot); after != before {
		t.Fatalf("sources.toml changed after unknown remove:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestSetDefaultUpdatesEnabledSource(t *testing.T) {
	stateRoot := withStateHome(t)
	publicRepo := writeSourceRepo(t, "public", "")
	privateRepo := writeSourceRepo(t, "private", "")
	writeSources(t, stateRoot, stateFile("public",
		sourceBlock("public", publicRepo, nil)+
			sourceBlock("private", privateRepo, nil),
	))

	registry, source, ok, err := SetDefault("private")
	if err != nil {
		t.Fatalf("SetDefault() error = %v, want nil", err)
	}
	if !ok {
		t.Fatal("SetDefault() ok = false, want true")
	}
	if source.ID != "private" || !source.Enabled {
		t.Fatalf("SetDefault() source = %#v, want enabled private", source)
	}
	if registry.Default != "private" {
		t.Fatalf("SetDefault() default = %q, want private", registry.Default)
	}
	if !strings.Contains(readSourcesFile(t, stateRoot), "default = \"private\"\n") {
		t.Fatalf("sources.toml after SetDefault(private) =\n%s\nwant private default", readSourcesFile(t, stateRoot))
	}
}

func TestSetDefaultUnknownOrDisabledLeavesStateUnchanged(t *testing.T) {
	stateRoot := withStateHome(t)
	publicRepo := writeSourceRepo(t, "public", "")
	writeSources(t, stateRoot, stateFile("public",
		sourceBlock("public", publicRepo, nil)+
			sourceBlock("disabled", filepath.Join(t.TempDir(), "disabled"), new(false)),
	))
	before := readSourcesFile(t, stateRoot)

	for _, id := range []string{"missing", "disabled"} {
		if _, _, ok, err := SetDefault(id); err != nil {
			t.Fatalf("SetDefault(%q) error = %v, want nil", id, err)
		} else if ok {
			t.Fatalf("SetDefault(%q) ok = true, want false", id)
		}
		if after := readSourcesFile(t, stateRoot); after != before {
			t.Fatalf("sources.toml changed after SetDefault(%q):\nbefore:\n%s\nafter:\n%s", id, before, after)
		}
	}
}

func readSourcesFile(t *testing.T, stateRoot string) string {
	t.Helper()

	contents, err := os.ReadFile(filepath.Join(stateRoot, "tuck", "sources.toml"))
	if err != nil {
		t.Fatalf("read sources.toml: %v", err)
	}
	return string(contents)
}

func quote(s string) string {
	return strconv.Quote(s)
}
