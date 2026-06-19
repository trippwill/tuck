package plan

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trippwill/tuck/internal/target"
)

func TestActionTypeJSONPreservesStringValues(t *testing.T) {
	got, err := json.Marshal(Action{
		Type: ActionRemoveSymlink,
		Path: "/tmp/link",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := `{"type":"remove_symlink","path":"/tmp/link"}`
	if string(got) != want {
		t.Fatalf("Marshal() = %s, want %s", got, want)
	}
}

func TestConflictCodeJSONPreservesStringValues(t *testing.T) {
	got, err := json.Marshal(Conflict{
		Code:    target.ConflictPathMismatch,
		Path:    "/tmp/link",
		Message: "path escapes root",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := `{"code":"path_mismatch","path":"/tmp/link","message":"path escapes root"}`
	if string(got) != want {
		t.Fatalf("Marshal() = %s, want %s", got, want)
	}
}

func TestTargetConflictCodeValues(t *testing.T) {
	tests := map[string]target.ConflictCode{
		"absent":                target.ConflictAbsent,
		"conflict":              target.ConflictGeneric,
		"inside_source_repo":    target.ConflictInsideSourceRepo,
		"multiple_providers":    target.ConflictMultipleProviders,
		"not_a_managed_symlink": target.ConflictNotManagedSymlink,
		"outside_target_root":   target.ConflictOutsideTargetRoot,
		"package_path_exists":   target.ConflictPackagePathExists,
		"path_mismatch":         target.ConflictPathMismatch,
		"real_directory":        target.ConflictRealDirectory,
		"real_file":             target.ConflictRealFile,
		"special_file":          target.ConflictSpecialFile,
		"unmanaged_symlink":     target.ConflictUnmanagedSymlink,
		"owned_by_other":        target.ConflictOwnedByOther,
	}
	for want, code := range tests {
		if string(code) != want {
			t.Fatalf("conflict code = %q, want %q", code, want)
		}
	}
}

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

	plan := Plan{Actions: []Action{
		{Type: ActionMkdir, Path: filepath.Dir(target)},
		{Type: ActionSymlink, LinkPath: target, Payload: "../../../src/zsh/.config/zsh/.zshrc", Target: source},
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

	plan := Plan{Actions: []Action{{Type: ActionMove, Src: src, Dst: dst}}}
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

	if err := Apply(Plan{Actions: []Action{{Type: ActionRmdir, Path: dir}}}); err != nil {
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

	if err := Apply(Plan{Actions: []Action{{Type: ActionRmdir, Path: dir}}}); err == nil {
		t.Fatal("Apply() error = nil, want error")
	}
}

func TestApplyMoveReportsErrors(t *testing.T) {
	root := t.TempDir()
	err := Apply(Plan{Actions: []Action{{
		Type: ActionMove,
		Src:  filepath.Join(root, "missing"),
		Dst:  filepath.Join(root, "dst"),
	}}})
	if err == nil {
		t.Fatal("Apply() error = nil, want error")
	}
}

func TestApplyRejectsUnknownActionType(t *testing.T) {
	err := Apply(Plan{Actions: []Action{{Type: ActionType("bogus")}}})
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

	err := Apply(Plan{Actions: []Action{
		{Type: ActionMove, Src: src, Dst: dst},
		{Type: ActionSymlink, LinkPath: filepath.Join(blockingFile, "link"), Payload: "payload"},
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

	err := Apply(Plan{Actions: []Action{
		{Type: ActionRemoveSymlink, Path: target},
		{Type: ActionMove, Src: packagePath, Dst: target},
		{Type: ActionRmdir, Path: packageDir},
		{Type: ActionRmdir, Path: packageParent},
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
