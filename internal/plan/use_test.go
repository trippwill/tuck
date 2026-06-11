package plan

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestApplyMovesFile(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "home/file")
	dst := filepath.Join(root, "src/pkg/file")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("contents"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := UsePlan{Actions: []Action{{Type: "move", Src: src, Dst: dst}}}
	if err := Apply(plan); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if _, err := os.Lstat(src); !os.IsNotExist(err) {
		t.Fatalf("source exists after move, err = %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != "contents" {
		t.Fatalf("moved contents = %q", got)
	}
}

func TestApplyRemovesEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src/pkg/.config/zsh")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Apply(UsePlan{Actions: []Action{{Type: "rmdir", Path: dir}}}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		t.Fatalf("directory exists after rmdir, err = %v", err)
	}
}

func TestApplyRmdirReportsNonEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src/pkg/.config/zsh")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file"), []byte("contents"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Apply(UsePlan{Actions: []Action{{Type: "rmdir", Path: dir}}}); err == nil {
		t.Fatal("Apply() error = nil, want error")
	}
}

func TestApplyMoveReportsErrors(t *testing.T) {
	root := t.TempDir()
	err := Apply(UsePlan{Actions: []Action{{
		Type: "move",
		Src:  filepath.Join(root, "missing"),
		Dst:  filepath.Join(root, "dst"),
	}}})
	if err == nil {
		t.Fatal("Apply() error = nil, want error")
	}
}

func TestApplyRejectsUnknownActionType(t *testing.T) {
	err := Apply(UsePlan{Actions: []Action{{Type: "bogus"}}})
	if err == nil {
		t.Fatal("Apply() error = nil, want error")
	}
	if !errors.Is(err, ErrApply) {
		t.Fatalf("Apply() error = %v, want errors.Is(..., %v)", err, ErrApply)
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("Apply() error = %q, want unknown action type", err.Error())
	}
}
