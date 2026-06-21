package status

import (
	"testing"

	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/state"
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

func TestEntryFromCopyClassReportsCopyStates(t *testing.T) {
	tests := map[string]struct {
		class     target.CopyClass
		wantState string
		wantCode  target.ConflictCode
	}{
		"absent": {
			class:     target.CopyClass{Kind: target.CopyAbsent},
			wantState: StateAbsent,
		},
		"tracked missing": {
			class:     target.CopyClass{Kind: target.CopyTrackedAbsent, Message: "tracked copied target is missing"},
			wantState: StateCopyMissing,
			wantCode:  target.ConflictGeneric,
		},
		"unchanged": {
			class:     target.CopyClass{Kind: target.CopyUnchanged},
			wantState: StateDeployed,
		},
		"source changed": {
			class:     target.CopyClass{Kind: target.CopySourceChanged, Message: "copied source changed"},
			wantState: string(target.CopySourceChanged),
			wantCode:  target.ConflictCopySourceChanged,
		},
		"target changed": {
			class:     target.CopyClass{Kind: target.CopyTargetChanged, Message: "copied target changed"},
			wantState: string(target.CopyTargetChanged),
			wantCode:  target.ConflictCopyTargetChanged,
		},
		"both changed": {
			class:     target.CopyClass{Kind: target.CopyBothChanged, Message: "copied source and target both changed"},
			wantState: string(target.CopyBothChanged),
			wantCode:  target.ConflictCopyDrift,
		},
		"untracked": {
			class:     target.CopyClass{Kind: target.CopyUntracked, Message: "target is an untracked real file"},
			wantState: StateConflict,
			wantCode:  target.ConflictRealFile,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			entry := entryFromCopyClass("/home/me/config", "public:home:app", "config", tt.class)
			if entry.State != tt.wantState {
				t.Fatalf("state = %q, want %q", entry.State, tt.wantState)
			}
			if entry.Code != tt.wantCode {
				t.Fatalf("code = %q, want %q", entry.Code, tt.wantCode)
			}
			if entry.Message != tt.class.Message {
				t.Fatalf("message = %q, want %q", entry.Message, tt.class.Message)
			}
		})
	}
}

func TestEntryFromCopyClassReportsCopyOwnedByOther(t *testing.T) {
	class := target.CopyClass{
		Kind:    target.CopyOwnedOther,
		Message: "target is a copied file managed by public:home:other",
		Owner: target.Owner{
			Identity: packages.Identity{
				Source:  "public",
				Context: packages.ContextHome,
				Name:    "other",
				Root:    "/src/other",
			},
			PackageRel: "config",
			EntryPath:  "/src/other/config",
		},
		Record: state.Copy{Package: "other"},
	}

	entry := entryFromCopyClass("/home/me/config", "public:home:app", "config", class)
	if entry.State != StateOwnedByOther {
		t.Fatalf("state = %q, want %q", entry.State, StateOwnedByOther)
	}
	if entry.Code != target.ConflictOwnedByOther {
		t.Fatalf("code = %q, want %q", entry.Code, target.ConflictOwnedByOther)
	}
	if entry.Owner != "public:home:other" {
		t.Fatalf("owner = %q, want public:home:other", entry.Owner)
	}
}
