package packages

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
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
