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

func TestApplyPreflightsBeforeMutating(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "home/file")
	dst := filepath.Join(root, "src/pkg/file")
	blockingFile := filepath.Join(root, "home/blocked")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("contents"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blockingFile, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Apply(UsePlan{Actions: []Action{
		{Type: "move", Src: src, Dst: dst},
		{Type: "symlink", LinkPath: filepath.Join(blockingFile, "link"), Payload: "payload"},
	}})
	if err == nil {
		t.Fatal("Apply() error = nil, want error")
	}
	if !errors.Is(err, ErrApply) {
		t.Fatalf("Apply() error = %v, want errors.Is(..., %v)", err, ErrApply)
	}
	if _, err := os.Lstat(src); err != nil {
		t.Fatalf("source was mutated before preflight failure: %v", err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatalf("destination exists after preflight failure, err = %v", err)
	}
}

func TestApplyPreflightAccountsForPlannedState(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "home/config")
	packagePath := filepath.Join(root, "src/pkg/.config/app/config")
	packageDir := filepath.Dir(packagePath)
	packageParent := filepath.Dir(packageDir)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(packagePath, []byte("contents"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(packagePath, target); err != nil {
		t.Fatal(err)
	}

	err := Apply(UsePlan{Actions: []Action{
		{Type: "remove_symlink", Path: target},
		{Type: "move", Src: packagePath, Dst: target},
		{Type: "rmdir", Path: packageDir},
		{Type: "rmdir", Path: packageParent},
	}})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != "contents" {
		t.Fatalf("moved contents = %q", got)
	}
	if _, err := os.Lstat(packageDir); !os.IsNotExist(err) {
		t.Fatalf("package dir exists after prune, err = %v", err)
	}
	if _, err := os.Lstat(packageParent); !os.IsNotExist(err) {
		t.Fatalf("package parent exists after prune, err = %v", err)
	}
}
