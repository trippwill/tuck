package target

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pathutil"
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

func InferOwner(linkPath string, source state.Source, context string, targetRoot string) (Owner, bool) {
	payload, err := os.Readlink(linkPath)
	if err != nil {
		return Owner{}, false
	}
	targetAbs := payload
	if !filepath.IsAbs(payload) {
		targetAbs = filepath.Join(filepath.Dir(linkPath), payload)
	}
	targetAbs = filepath.Clean(targetAbs)
	base := filepath.Clean(packages.Base(source, context))
	if !pathutil.Inside(targetAbs, base) {
		return Owner{}, false
	}
	relToBase, err := pathutil.RelInside(base, targetAbs)
	if err != nil {
		return Owner{}, false
	}
	parts := strings.Split(relToBase, string(filepath.Separator))
	if len(parts) < 2 {
		return Owner{}, false
	}
	pkgName := parts[0]
	packageRoot := filepath.Join(base, pkgName)
	packageRel, err := pathutil.RelInside(packageRoot, targetAbs)
	if err != nil {
		return Owner{}, false
	}
	expectedTarget := filepath.Clean(filepath.Join(targetRoot, packageRel))
	return Owner{
		Identity: packages.Identity{
			Source:  source.ID,
			Context: context,
			Name:    pkgName,
			Root:    packageRoot,
		},
		PackageRel:     packageRel,
		EntryPath:      targetAbs,
		ExpectedTarget: expectedTarget,
		Mismatch:       filepath.Clean(linkPath) != expectedTarget,
	}, true
}

func (c Class) ConflictCode() string {
	switch c.Kind {
	case RealDirectory:
		return "real_directory"
	case RealFile:
		return "real_file"
	case SpecialFile:
		return "special_file"
	case UnmanagedSymlink:
		return "unmanaged_symlink"
	case ManagedOther:
		return "owned_by_other"
	case PathMismatch:
		return "path_mismatch"
	default:
		return "conflict"
	}
}
