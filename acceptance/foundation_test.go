//go:build tuck_testhooks

package acceptance

import (
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestFoundation(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:   "testdata/script/foundation",
		Setup: setupScriptEnv,
		Cmds:  scriptCommands(),
	})
}
