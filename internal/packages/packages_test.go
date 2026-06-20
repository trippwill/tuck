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

func TestResolveAppliesPackageFileConfigAndSkipsManifest(t *testing.T) {
	root := t.TempDir()
	sourceRoot := filepath.Join(root, "src")
	packageRoot := filepath.Join(sourceRoot, "app")
	writePackageTestFile(t, filepath.Join(sourceRoot, manifest.ManifestFilename), "name = \"public\"\n")
	writePackageTestFile(t, filepath.Join(packageRoot, ".config/app/config"), "config")
	writePackageTestFile(t, filepath.Join(packageRoot, manifest.ManifestFilename), `[[file]]
path = ".config/app/config"
deploy = "copy"
mode = "0600"
`)
	source := state.Source{ID: "public", Path: sourceRoot}

	got, err := ResolveOne(source, ContextHome, "app")
	if err != nil {
		t.Fatalf("ResolveOne() error = %v, want nil", err)
	}
	leaves := Leaves(got.Entries)
	if len(leaves) != 1 {
		t.Fatalf("leaves = %#v, want only package config leaf", leaves)
	}
	if leaves[0].Rel != ".config/app/config" || leaves[0].Deploy != DeployCopy || leaves[0].Mode != "0600" {
		t.Fatalf("configured leaf = %#v, want copy mode 0600", leaves[0])
	}
}

func TestLoadConfigRejectsInvalidFilePath(t *testing.T) {
	root := t.TempDir()
	writePackageTestFile(t, filepath.Join(root, "config"), "config")
	writePackageTestFile(t, filepath.Join(root, manifest.ManifestFilename), `[[file]]
path = "../config"
deploy = "copy"
`)

	_, err := LoadConfig(root, []Entry{{Path: filepath.Join(root, "config"), Rel: "config"}})
	if !errors.Is(err, ErrConfigInvalid) {
		t.Fatalf("LoadConfig() error = %v, want ErrConfigInvalid", err)
	}
}

func TestNormalizeModeFlagAcceptsChmodExpressions(t *testing.T) {
	got, err := NormalizeModeFlag("u=rw,go=", "0644")
	if err != nil {
		t.Fatalf("NormalizeModeFlag() error = %v, want nil", err)
	}
	if got != "0600" {
		t.Fatalf("NormalizeModeFlag() = %q, want 0600", got)
	}
	got, err = NormalizeModeFlag("g+x", "0600")
	if err != nil {
		t.Fatalf("NormalizeModeFlag() error = %v, want nil", err)
	}
	if got != "0610" {
		t.Fatalf("NormalizeModeFlag() = %q, want 0610", got)
	}
}

func directoryRels(entries []Entry) []string {
	rels := make([]string, 0, len(entries))
	for _, entry := range entries {
		rels = append(rels, entry.Rel)
	}
	return rels
}

func writePackageTestFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
