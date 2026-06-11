package packages

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

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
	for _, file := range []string{"README", "tuck.toml"} {
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

func TestResolveForAdoptDoesNotRequireExistingPackage(t *testing.T) {
	source := state.Source{ID: "public", Path: "/src"}

	got, err := ResolveForAdopt(source, ContextHome, "nvim")
	if err != nil {
		t.Fatalf("ResolveForAdopt() error = %v", err)
	}
	if got.String() != "public:home:nvim" {
		t.Fatalf("ResolveForAdopt().String() = %q", got.String())
	}
	if got.Root != filepath.Join("/src", "nvim") {
		t.Fatalf("ResolveForAdopt().Root = %q", got.Root)
	}
}

func TestResolveForAdoptValidatesPackageRef(t *testing.T) {
	_, err := ResolveForAdopt(state.Source{ID: "public", Path: "/src"}, ContextHome, "bad/ref")
	if !errors.Is(err, pkgref.ErrInvalidRef) {
		t.Fatalf("ResolveForAdopt() error = %v, want ErrInvalidRef", err)
	}
}
