package target

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/state"
)

type CopyKind string

const (
	CopyAbsent        CopyKind = "absent"
	CopyTrackedAbsent CopyKind = "copy_missing"
	CopyUnchanged     CopyKind = "copy_unchanged"
	CopySourceChanged CopyKind = "copy_source_modified"
	CopyTargetChanged CopyKind = "copy_target_modified"
	CopyBothChanged   CopyKind = "copy_drift"
	CopyUntracked     CopyKind = "copy_untracked"
	CopyOwnedOther    CopyKind = "copy_owned_by_other"
	CopySpecial       CopyKind = "copy_conflict"
)

type CopyClass struct {
	Kind    CopyKind
	Message string
	Record  state.Copy
	Owner   Owner
}

func ClassifyCopy(entry PackageEntry, registry state.Registry) CopyClass {
	record, tracked := registry.CopyByEntry(entry.Identity.Source, entry.Identity.Context, entry.Identity.Name, entry.Entry.Rel)
	info, err := os.Lstat(entry.PhysicalPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if tracked {
				return CopyClass{Kind: CopyTrackedAbsent, Record: record, Message: "tracked copied target is missing"}
			}
			return CopyClass{Kind: CopyAbsent}
		}
		return CopyClass{Kind: CopySpecial, Record: record, Message: err.Error()}
	}
	if !info.Mode().IsRegular() {
		return CopyClass{Kind: CopySpecial, Record: record, Message: "target is not a regular file"}
	}
	if !tracked {
		if other, ok := registry.CopyByTarget(entry.Identity.Source, entry.Identity.Context, entry.TargetPath); ok {
			return CopyClass{Kind: CopyOwnedOther, Record: other, Message: "target is a copied file managed by " + ownerString(other), Owner: ownerFromCopy(other, filepath.Join(filepath.Dir(entry.Identity.Root), other.Package))}
		}
		return CopyClass{Kind: CopyUntracked, Message: "target is an untracked real file"}
	}

	sourceChecksum, err := state.FileChecksum(entry.Entry.Path)
	if err != nil {
		return CopyClass{Kind: CopySpecial, Record: record, Message: err.Error()}
	}
	targetChecksum, err := state.FileChecksum(entry.PhysicalPath)
	if err != nil {
		return CopyClass{Kind: CopySpecial, Record: record, Message: err.Error()}
	}
	expectedMode, err := expectedMode(entry)
	if err != nil {
		return CopyClass{Kind: CopySpecial, Record: record, Message: err.Error()}
	}
	targetMode := modeString(info.Mode().Perm())
	sourceChanged := sourceChecksum != record.SourceChecksum || expectedMode != record.TargetMode
	targetChanged := targetChecksum != record.TargetChecksum || targetMode != record.TargetMode
	class := CopyClass{Kind: CopyUnchanged, Record: record}
	switch {
	case sourceChanged && targetChanged:
		class.Kind = CopyBothChanged
		class.Message = "copied source and target both changed"
	case sourceChanged:
		class.Kind = CopySourceChanged
		class.Message = "copied source changed"
	case targetChanged:
		class.Kind = CopyTargetChanged
		class.Message = "copied target changed"
	}
	return class
}

func (c CopyClass) ConflictCode() ConflictCode {
	switch c.Kind {
	case CopySourceChanged:
		return ConflictCopySourceChanged
	case CopyTargetChanged:
		return ConflictCopyTargetChanged
	case CopyBothChanged:
		return ConflictCopyDrift
	case CopyOwnedOther:
		return ConflictOwnedByOther
	case CopyUntracked:
		return ConflictRealFile
	default:
		return ConflictGeneric
	}
}

func expectedMode(entry PackageEntry) (string, error) {
	if entry.Entry.Mode != "" {
		return entry.Entry.Mode, nil
	}
	return packages.ModeFromFile(entry.Entry.Path)
}

func modeString(mode fs.FileMode) string {
	return fmt.Sprintf("%04o", mode.Perm())
}

func ownerString(record state.Copy) string {
	return record.Source + ":" + record.Context + ":" + record.Package
}

func ownerFromCopy(record state.Copy, packageRoot string) Owner {
	return Owner{
		Identity: packages.Identity{
			Source:  record.Source,
			Context: record.Context,
			Name:    record.Package,
			Root:    packageRoot,
		},
		PackageRel:     record.Path,
		EntryPath:      filepath.Join(packageRoot, record.Path),
		ExpectedTarget: record.Target,
	}
}
