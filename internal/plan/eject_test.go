package plan

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestPruneAfterEjectPlansEmptyParentsBelowPackageRoot(t *testing.T) {
	root := t.TempDir()
	packageRoot := filepath.Join(root, "zsh")
	packagePath := filepath.Join(packageRoot, ".config/zsh/.zshrc")
	writeTestFile(t, packagePath)

	got, err := pruneAfterEject(packageRoot, packagePath)
	if err != nil {
		t.Fatalf("pruneAfterEject() error = %v", err)
	}
	want := []string{
		filepath.Join(packageRoot, ".config/zsh"),
		filepath.Join(packageRoot, ".config"),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("pruneAfterEject() = %#v, want %#v", got, want)
	}
}

func TestPruneAfterEjectLeavesPackageRoot(t *testing.T) {
	root := t.TempDir()
	packagePath := filepath.Join(root, "zsh/.zshrc")
	writeTestFile(t, packagePath)

	got, err := pruneAfterEject(filepath.Join(root, "zsh"), packagePath)
	if err != nil {
		t.Fatalf("pruneAfterEject() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("pruneAfterEject() = %#v, want no package root removal", got)
	}
}

func TestPruneAfterEjectStopsAtSiblingLeaf(t *testing.T) {
	root := t.TempDir()
	packageRoot := filepath.Join(root, "zsh")
	packagePath := filepath.Join(packageRoot, ".config/zsh/.zshrc")
	writeTestFile(t, packagePath)
	writeTestFile(t, filepath.Join(packageRoot, ".config/zsh/.zprofile"))

	got, err := pruneAfterEject(packageRoot, packagePath)
	if err != nil {
		t.Fatalf("pruneAfterEject() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("pruneAfterEject() = %#v, want no removals", got)
	}
}

func TestPruneAfterEjectStopsAtExtraDirectory(t *testing.T) {
	root := t.TempDir()
	packageRoot := filepath.Join(root, "zsh")
	packagePath := filepath.Join(packageRoot, ".config/zsh/.zshrc")
	writeTestFile(t, packagePath)
	writeTestFile(t, filepath.Join(packageRoot, ".config/zsh/plugins/README"))

	got, err := pruneAfterEject(packageRoot, packagePath)
	if err != nil {
		t.Fatalf("pruneAfterEject() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("pruneAfterEject() = %#v, want no removals", got)
	}
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("contents"), 0o644); err != nil {
		t.Fatal(err)
	}
}
