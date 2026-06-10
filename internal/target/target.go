package target

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/state"
)

type Kind string

const (
	Absent           Kind = "absent"
	RealDirectory    Kind = "real_directory"
	RealFile         Kind = "real_file"
	SpecialFile      Kind = "special_file"
	UnmanagedSymlink Kind = "unmanaged_symlink"
	Managed          Kind = "managed"
	ManagedSelected  Kind = "managed_selected"
	ManagedOther     Kind = "managed_other"
	PathMismatch     Kind = "path_mismatch"
)

type Owner struct {
	Identity       packages.Identity
	PackageRel     string
	EntryPath      string
	ExpectedTarget string
	Mismatch       bool
}

type Class struct {
	Kind    Kind
	Message string
	Owner   Owner
}

func Classify(targetPath string, source state.Source, context string, targetRoot string, selected *packages.Identity, selectedRel string) Class {
	info, err := os.Lstat(targetPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Class{Kind: Absent}
		}
		return Class{Kind: SpecialFile, Message: err.Error()}
	}
	if info.Mode()&os.ModeSymlink == 0 {
		if info.IsDir() {
			return Class{Kind: RealDirectory, Message: "target is a real directory"}
		}
		if info.Mode().IsRegular() {
			return Class{Kind: RealFile, Message: "target is a real file"}
		}
		return Class{Kind: SpecialFile, Message: "target is a special file"}
	}

	owner, ok := InferOwner(targetPath, source, context, targetRoot)
	if !ok {
		return Class{Kind: UnmanagedSymlink, Message: "target is an unmanaged symlink"}
	}
	if owner.Mismatch {
		return Class{Kind: PathMismatch, Message: "managed symlink path does not match package entry", Owner: owner}
	}
	if selected == nil {
		return Class{Kind: Managed, Owner: owner}
	}
	if owner.Identity.Name == selected.Name && owner.PackageRel == selectedRel {
		return Class{Kind: ManagedSelected, Owner: owner}
	}
	return Class{Kind: ManagedOther, Message: fmt.Sprintf("target is managed by %s", owner.Identity.String()), Owner: owner}
}
