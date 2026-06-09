//go:build tuck_testhooks

package acceptance

import (
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestStatus(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:   "testdata/script/status",
		Setup: setupScriptEnv,
		Cmds:  scriptCommands(),
	})
}
