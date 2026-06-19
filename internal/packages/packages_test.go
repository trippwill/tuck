package packages

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/pkgref"
	"github.com/trippwill/tuck/internal/state"
)

func TestDiscoverSkipsDotPrefixedDirectories(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"zsh", "git", ".root", ".hidden"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{"README", manifest.ManifestFilename} {
		if err := os.WriteFile(filepath.Join(root, file), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	want := []string{"git", "zsh"}
	if !slices.Equal(got, want) {
		t.Fatalf("Discover() = %#v, want %#v", got, want)
	}
}

func TestDirectoriesReturnsOnlyDirectoriesWithoutChildDirectories(t *testing.T) {
	entries := []Entry{
		{Rel: ".config", Dir: true},
		{Rel: ".config/app", Dir: true},
		{Rel: ".config/app/config", Dir: false},
		{Rel: ".config/app/plugins", Dir: true},
		{Rel: ".config/app/plugins/plugin.toml", Dir: false},
		{Rel: ".ssh", Dir: true},
		{Rel: ".ssh/config", Dir: false},
		{Rel: "bin", Dir: true},
		{Rel: "bin/tool", Dir: false},
	}

	got := directoryRels(Directories(entries))
	want := []string{".config/app/plugins", ".ssh", "bin"}
	if !slices.Equal(got, want) {
		t.Fatalf("Directories() = %#v, want %#v", got, want)
	}
}

func TestDirectoriesKeepsSiblingPrefixDirectoriesIndependent(t *testing.T) {
	entries := []Entry{
		{Rel: "config", Dir: true},
		{Rel: "config-extra", Dir: true},
		{Rel: "config/app", Dir: true},
		{Rel: "config/app/settings.toml", Dir: false},
	}

	got := directoryRels(Directories(entries))
	want := []string{"config-extra", "config/app"}
	if !slices.Equal(got, want) {
		t.Fatalf("Directories() = %#v, want %#v", got, want)
	}
}

func TestResolveIdentityDoesNotRequireExistingPackage(t *testing.T) {
	source := state.Source{ID: "public", Path: "/src"}

	got, err := ResolveIdentity(source, ContextHome, "nvim")
	if err != nil {
		t.Fatalf("ResolveIdentity() error = %v", err)
	}
	if got.String() != "public:home:nvim" {
		t.Fatalf("ResolveIdentity().String() = %q", got.String())
	}
	if got.Root != filepath.Join("/src", "nvim") {
		t.Fatalf("ResolveIdentity().Root = %q", got.Root)
	}
}

func TestResolveIdentityValidatesPackageRef(t *testing.T) {
	_, err := ResolveIdentity(state.Source{ID: "public", Path: "/src"}, ContextHome, "bad/ref")
	if !errors.Is(err, pkgref.ErrInvalidRef) {
		t.Fatalf("ResolveIdentity() error = %v, want ErrInvalidRef", err)
	}
}

func directoryRels(entries []Entry) []string {
	rels := make([]string, 0, len(entries))
	for _, entry := range entries {
		rels = append(rels, entry.Rel)
	}
	return rels
}
