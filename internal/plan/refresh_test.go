package plan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/state"
)

func TestBuildRefreshRecreatesSelectedManagedSymlink(t *testing.T) {
	home, source := setupRefreshSource(t)
	packagePath := filepath.Join(source, "zsh/.config/zsh/.zshrc")
	targetPath := filepath.Join(home, ".config/zsh/.zshrc")
	writeRefreshFile(t, packagePath)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(packagePath, targetPath); err != nil {
		t.Fatal(err)
	}

	got, err := BuildRefresh(RefreshOptions{Refs: []string{"zsh"}})
	if err != nil {
		t.Fatalf("BuildRefresh() error = %v, want nil", err)
	}
	if len(got.Actions) != 2 {
		t.Fatalf("BuildRefresh() actions = %#v, want unlink and link", got.Actions)
	}
	if got.Actions[0].Type != ActionRemoveSymlink || got.Actions[0].Path != targetPath {
		t.Fatalf("first action = %#v, want remove_symlink for target", got.Actions[0])
	}
	if got.Actions[1].Type != ActionSymlink || got.Actions[1].LinkPath != targetPath {
		t.Fatalf("second action = %#v, want symlink for target", got.Actions[1])
	}
	if got.Actions[1].Payload != "../../../src/zsh/.config/zsh/.zshrc" {
		t.Fatalf("refresh payload = %q, want normalized relative payload", got.Actions[1].Payload)
	}
}

func TestBuildRefreshAccumulatesConflictsAndSuppressesActions(t *testing.T) {
	home, source := setupRefreshSource(t)
	writeRefreshFile(t, filepath.Join(source, "zsh/.config/zsh/.zshrc"))
	writeRefreshFile(t, filepath.Join(source, "zsh/.config/zsh/.zprofile"))
	targetDir := filepath.Join(home, ".config/zsh")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeRefreshFile(t, filepath.Join(targetDir, ".zshrc"))
	if err := os.Symlink("../outside", filepath.Join(targetDir, ".zprofile")); err != nil {
		t.Fatal(err)
	}

	got, err := BuildRefresh(RefreshOptions{Refs: []string{"zsh"}, Apply: true})
	if err != nil {
		t.Fatalf("BuildRefresh() error = %v, want nil plan with conflicts", err)
	}
	if len(got.Conflicts) != 2 {
		t.Fatalf("BuildRefresh() conflicts = %#v, want two conflicts", got.Conflicts)
	}
	if len(got.Actions) != 0 {
		t.Fatalf("BuildRefresh() actions = %#v, want none when conflicts exist", got.Actions)
	}
}

func setupRefreshSource(t *testing.T) (home string, source string) {
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
	writeRefreshFile(t, filepath.Join(source, manifest.ManifestFilename), "name = \"public\"\n")
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

func writeRefreshFile(t *testing.T, path string, contents ...string) {
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
