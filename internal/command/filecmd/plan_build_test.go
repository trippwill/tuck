package filecmd

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/state"
	"github.com/trippwill/tuck/internal/target"
)

func TestBuildAdoptReplaceOverwritesPackageFile(t *testing.T) {
	home, source := setupAdoptSource(t)
	packagePath := filepath.Join(source, "zsh/.zshrc")
	targetPath := filepath.Join(home, ".zshrc")
	writeTestFile(t, packagePath, "old")
	writeTestFile(t, targetPath, "new")
	if err := os.Chmod(targetPath, 0o600); err != nil {
		t.Fatal(err)
	}

	blocked, err := buildAdopt(AdoptRequest{File: targetPath, Ref: "zsh"})
	if err != nil {
		t.Fatalf("buildAdopt() error = %v", err)
	}
	if len(blocked.Conflicts) != 1 || blocked.Conflicts[0].Code != target.ConflictPackagePathExists {
		t.Fatalf("buildAdopt() conflicts = %#v, want package_path_exists", blocked.Conflicts)
	}

	got, err := buildAdopt(AdoptRequest{File: targetPath, Ref: "zsh", Replace: true, Apply: true})
	if err != nil {
		t.Fatalf("buildAdopt(--replace --apply) error = %v", err)
	}
	if !got.Applied || len(got.Conflicts) != 0 {
		t.Fatalf("buildAdopt(--replace --apply) = %#v, want applied plan without conflicts", got)
	}
	if got := readTestFile(t, packagePath); got != "new" {
		t.Fatalf("package contents = %q, want target contents", got)
	}
	info, err := os.Lstat(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("target mode = %v, want symlink", info.Mode())
	}
	sourceInfo, err := os.Stat(packagePath)
	if err != nil {
		t.Fatal(err)
	}
	if sourceInfo.Mode().Perm() != 0o600 {
		t.Fatalf("package mode = %v, want adopted target mode 0600", sourceInfo.Mode().Perm())
	}
}

func TestBuildAdoptReplaceCopyModeOverwritesPackageFile(t *testing.T) {
	home, source := setupAdoptSource(t)
	packagePath := filepath.Join(source, "zsh/.zshrc")
	targetPath := filepath.Join(home, ".zshrc")
	writeTestFile(t, packagePath, "old")
	writeTestFile(t, targetPath, "new")
	if err := state.Save(state.Registry{
		Default: "public",
		Sources: []state.Source{{
			ID:      "public",
			Path:    source,
			Enabled: true,
		}},
		Copies: []state.Copy{{
			Source:  "public",
			Context: "home",
			Package: "old",
			Path:    ".zshrc",
			Target:  targetPath,
		}},
	}); err != nil {
		t.Fatalf("state.Save() error = %v", err)
	}

	got, err := buildAdopt(AdoptRequest{File: targetPath, Ref: "zsh", Copy: true, Replace: true, Apply: true})
	if err != nil {
		t.Fatalf("buildAdopt(--copy --replace --apply) error = %v", err)
	}
	if !got.Applied || len(got.Conflicts) != 0 {
		t.Fatalf("buildAdopt(--copy --replace --apply) = %#v, want applied plan without conflicts", got)
	}
	if got := readTestFile(t, packagePath); got != "new" {
		t.Fatalf("package contents = %q, want target contents", got)
	}
	info, err := os.Lstat(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("target mode = %v, want regular copied file", info.Mode())
	}
	registry, err := state.Load()
	if err != nil {
		t.Fatal(err)
	}
	record, ok := registry.CopyByTarget("public", "home", targetPath)
	if !ok {
		t.Fatalf("copied target %q was not tracked", targetPath)
	}
	if record.Package != "zsh" || record.Path != ".zshrc" {
		t.Fatalf("copied target owner = %#v, want zsh/.zshrc", record)
	}
	if _, ok := registry.CopyByEntry("public", "home", "old", ".zshrc"); ok {
		t.Fatalf("old copied-file owner was not removed")
	}
}

func TestBuildAdoptReplaceRejectsNonRegularPackagePath(t *testing.T) {
	home, source := setupAdoptSource(t)
	packagePath := filepath.Join(source, "zsh/.zshrc")
	targetPath := filepath.Join(home, ".zshrc")
	writeTestFile(t, filepath.Join(source, "target"))
	if err := os.MkdirAll(filepath.Dir(packagePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../target", packagePath); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, targetPath)

	got, err := buildAdopt(AdoptRequest{File: targetPath, Ref: "zsh", Replace: true})
	if err != nil {
		t.Fatalf("buildAdopt(--replace) error = %v", err)
	}
	if len(got.Conflicts) != 1 || got.Conflicts[0].Code != target.ConflictPackagePathExists {
		t.Fatalf("buildAdopt(--replace) conflicts = %#v, want package_path_exists", got.Conflicts)
	}
	if len(got.Actions) != 0 {
		t.Fatalf("buildAdopt(--replace) actions = %#v, want none", got.Actions)
	}
}

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

func setupAdoptSource(t *testing.T) (home string, source string) {
	t.Helper()
	root := t.TempDir()
	home = filepath.Join(root, "home")
	source = filepath.Join(root, "src")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(source, manifest.ManifestFilename), "name = \"public\"\n")
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	if err := state.Save(state.Registry{
		Default: "public",
		Sources: []state.Source{{
			ID:      "public",
			Path:    source,
			Enabled: true,
		}},
	}); err != nil {
		t.Fatalf("state.Save() error = %v", err)
	}
	return home, source
}

func writeTestFile(t *testing.T, path string, contents ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "contents"
	if len(contents) > 0 {
		body = contents[0]
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}
