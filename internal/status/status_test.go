package status

import (
	"testing"

	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/target"
)

func TestEntryFromClassReportsMismatch(t *testing.T) {
	owner := target.Owner{
		Identity: packages.Identity{
			Source:  "public",
			Context: packages.ContextHome,
			Name:    "zsh",
			Root:    "/src/zsh",
		},
		EntryPath:      "/src/zsh/.config/zsh/.zprofile",
		ExpectedTarget: "/home/me/.config/zsh/.zprofile",
		Mismatch:       true,
	}

	entry := entryFromClass("/home/me/.config/zsh/.zshrc", "public:home:zsh", "/src/zsh/.config/zsh/.zshrc", target.Class{
		Kind:    target.PathMismatch,
		Message: "managed symlink path does not match package entry",
		Owner:   owner,
	}, true)

	if entry.State != StateMismatch {
		t.Fatalf("state = %q, want %q", entry.State, StateMismatch)
	}
	if entry.Code != target.ConflictPathMismatch {
		t.Fatalf("code = %q, want %q", entry.Code, target.ConflictPathMismatch)
	}
	if entry.Owner != "public:home:zsh" {
		t.Fatalf("owner = %q, want public:home:zsh", entry.Owner)
	}
	if entry.ExpectedTarget != "/home/me/.config/zsh/.zprofile" {
		t.Fatalf("expectedTarget = %q, want mismatch target", entry.ExpectedTarget)
	}
}

func TestEntryFromClassReportsOwnedByOther(t *testing.T) {
	owner := target.Owner{
		Identity: packages.Identity{
			Source:  "public",
			Context: packages.ContextHome,
			Name:    "zsh",
			Root:    "/src/zsh",
		},
		EntryPath: "/src/zsh/.config/zsh/.zshrc",
	}

	entry := entryFromClass("/home/me/.config/zsh/.zshrc", "public:home:zsh-alt", "/src/zsh-alt/.config/zsh/.zshrc", target.Class{
		Kind:    target.ManagedOther,
		Message: "target is managed by public:home:zsh",
		Owner:   owner,
	}, true)

	if entry.State != StateOwnedByOther {
		t.Fatalf("state = %q, want %q", entry.State, StateOwnedByOther)
	}
	if entry.Code != target.ConflictOwnedByOther {
		t.Fatalf("code = %q, want %q", entry.Code, target.ConflictOwnedByOther)
	}
	if entry.Package != "public:home:zsh-alt" {
		t.Fatalf("package = %q, want selected package", entry.Package)
	}
	if entry.Owner != "public:home:zsh" {
		t.Fatalf("owner = %q, want public:home:zsh", entry.Owner)
	}
}
