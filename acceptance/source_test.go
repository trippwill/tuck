//go:build tuck_testhooks

package acceptance

import (
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestSource(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:   "testdata/script/source",
		Setup: setupScriptEnv,
		Cmds:  scriptCommands(),
	})
}
