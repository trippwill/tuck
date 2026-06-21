package target

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/state"
)

func TestClassifyCopyDriftStates(t *testing.T) {
	tests := map[string]struct {
		sourceAfter  string
		targetAfter  string
		removeTarget bool
		wantKind     CopyKind
		wantCode     ConflictCode
	}{
		"unchanged": {
			wantKind: CopyUnchanged,
			wantCode: ConflictGeneric,
		},
		"source changed": {
			sourceAfter: "source v2",
			wantKind:    CopySourceChanged,
			wantCode:    ConflictCopySourceChanged,
		},
		"target changed": {
			targetAfter: "target v2",
			wantKind:    CopyTargetChanged,
			wantCode:    ConflictCopyTargetChanged,
		},
		"both changed": {
			sourceAfter: "source v2",
			targetAfter: "target v2",
			wantKind:    CopyBothChanged,
			wantCode:    ConflictCopyDrift,
		},
		"tracked target missing": {
			removeTarget: true,
			wantKind:     CopyTrackedAbsent,
			wantCode:     ConflictGeneric,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			entry := writeCopyEntry(t, "v1", "v1")
			registry := state.Registry{Copies: []state.Copy{copyRecord(t, entry)}}
			if tt.sourceAfter != "" {
				writeTargetFile(t, entry.Entry.Path, tt.sourceAfter)
			}
			if tt.targetAfter != "" {
				writeTargetFile(t, entry.PhysicalPath, tt.targetAfter)
			}
			if tt.removeTarget {
				if err := os.Remove(entry.PhysicalPath); err != nil {
					t.Fatal(err)
				}
			}

			got := ClassifyCopy(entry, registry)
			if got.Kind != tt.wantKind {
				t.Fatalf("ClassifyCopy() kind = %q, want %q", got.Kind, tt.wantKind)
			}
			if got.ConflictCode() != tt.wantCode {
				t.Fatalf("ConflictCode() = %q, want %q", got.ConflictCode(), tt.wantCode)
			}
		})
	}
}

func TestClassifyCopyUntrackedAndOwnedTargets(t *testing.T) {
	t.Run("absent", func(t *testing.T) {
		entry := writeCopyEntry(t, "v1", "")
		got := ClassifyCopy(entry, state.Registry{})
		if got.Kind != CopyAbsent {
			t.Fatalf("ClassifyCopy() kind = %q, want %q", got.Kind, CopyAbsent)
		}
	})

	t.Run("untracked real file", func(t *testing.T) {
		entry := writeCopyEntry(t, "v1", "target")
		got := ClassifyCopy(entry, state.Registry{})
		if got.Kind != CopyUntracked {
			t.Fatalf("ClassifyCopy() kind = %q, want %q", got.Kind, CopyUntracked)
		}
		if got.ConflictCode() != ConflictRealFile {
			t.Fatalf("ConflictCode() = %q, want %q", got.ConflictCode(), ConflictRealFile)
		}
	})

	t.Run("owned by another copy record", func(t *testing.T) {
		entry := writeCopyEntry(t, "v1", "target")
		registry := state.Registry{Copies: []state.Copy{{
			Source:  entry.Identity.Source,
			Context: entry.Identity.Context,
			Package: "other",
			Path:    "config",
			Target:  entry.TargetPath,
		}}}

		got := ClassifyCopy(entry, registry)
		if got.Kind != CopyOwnedOther {
			t.Fatalf("ClassifyCopy() kind = %q, want %q", got.Kind, CopyOwnedOther)
		}
		if got.Owner.Identity.Name != "other" {
			t.Fatalf("owner package = %q, want other", got.Owner.Identity.Name)
		}
		if got.ConflictCode() != ConflictOwnedByOther {
			t.Fatalf("ConflictCode() = %q, want %q", got.ConflictCode(), ConflictOwnedByOther)
		}
	})
}

func TestClassifyCopySpecialTarget(t *testing.T) {
	entry := writeCopyEntry(t, "v1", "")
	if err := os.MkdirAll(entry.PhysicalPath, 0o755); err != nil {
		t.Fatal(err)
	}

	got := ClassifyCopy(entry, state.Registry{})
	if got.Kind != CopySpecial {
		t.Fatalf("ClassifyCopy() kind = %q, want %q", got.Kind, CopySpecial)
	}
}

func writeCopyEntry(t *testing.T, sourceContents string, targetContents string) PackageEntry {
	t.Helper()

	root := t.TempDir()
	packageRoot := filepath.Join(root, "src", "app")
	sourcePath := filepath.Join(packageRoot, "config")
	targetPath := filepath.Join(root, "home", "config")
	writeTargetFile(t, sourcePath, sourceContents)
	if targetContents != "" {
		writeTargetFile(t, targetPath, targetContents)
	}
	identity := packages.Identity{Source: "public", Context: packages.ContextHome, Name: "app", Root: packageRoot}
	return PackageEntry{
		Identity:     identity,
		Entry:        packages.Entry{Path: sourcePath, Rel: "config", Deploy: packages.DeployCopy},
		PackageID:    identity.String(),
		ProviderKey:  identity.String() + ":config",
		TargetPath:   targetPath,
		PhysicalPath: targetPath,
	}
}

func copyRecord(t *testing.T, entry PackageEntry) state.Copy {
	t.Helper()

	sourceChecksum, err := state.FileChecksum(entry.Entry.Path)
	if err != nil {
		t.Fatal(err)
	}
	targetChecksum, err := state.FileChecksum(entry.PhysicalPath)
	if err != nil {
		t.Fatal(err)
	}
	return state.Copy{
		Source:         entry.Identity.Source,
		Context:        entry.Identity.Context,
		Package:        entry.Identity.Name,
		Path:           entry.Entry.Rel,
		Target:         entry.TargetPath,
		SourceChecksum: sourceChecksum,
		TargetChecksum: targetChecksum,
		TargetMode:     "0644",
	}
}

func writeTargetFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
