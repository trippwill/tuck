package target

import (
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/state"
)

type PackageEntry struct {
	Identity     packages.Identity
	Entry        packages.Entry
	PackageID    string
	ProviderKey  string
	TargetPath   string
	PhysicalPath string
}

func NewPackageEntry(pkg packages.Resolved, entry packages.Entry, logicalRoot string, physicalPath func(string) string) (PackageEntry, error) {
	targetPath, _, err := pathutil.PackageToTarget(pkg.Identity.Root, entry.Path, logicalRoot)
	packageID := pkg.Identity.String()
	return PackageEntry{
		Identity:     pkg.Identity,
		Entry:        entry,
		PackageID:    packageID,
		ProviderKey:  packageID + ":" + entry.Rel,
		TargetPath:   targetPath,
		PhysicalPath: physicalPath(targetPath),
	}, err
}

func (entry PackageEntry) Classify(source state.Source, context string, targetRoot string) Class {
	return ClassifyAt(entry.TargetPath, entry.PhysicalPath, source, context, targetRoot, &entry.Identity, entry.Entry.Rel)
}
