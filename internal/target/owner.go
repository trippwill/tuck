package target

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/state"
)

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
