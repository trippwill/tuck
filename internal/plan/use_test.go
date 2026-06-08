package plan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyCreatesRelativeSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "home/.config/zsh/.zshrc")
	source := filepath.Join(root, "src/zsh/.config/zsh/.zshrc")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("zsh"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := UsePlan{Actions: []Action{
		{Type: "mkdir", Path: filepath.Dir(target)},
		{Type: "symlink", LinkPath: target, Payload: "../../../src/zsh/.config/zsh/.zshrc", Target: source},
	}}
	if err := Apply(plan); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	got, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if got != "../../../src/zsh/.config/zsh/.zshrc" {
		t.Fatalf("payload = %q", got)
	}
}
