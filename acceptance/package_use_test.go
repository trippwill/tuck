//go:build tuck_testhooks

package acceptance

import (
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestPackageUse(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:   "testdata/script/package_use",
		Setup: setupScriptEnv,
		Cmds:  scriptCommands(),
	})
}
