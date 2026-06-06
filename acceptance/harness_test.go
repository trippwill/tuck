//go:build tuck_testhooks

package acceptance

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/trippwill/tuck/internal/app"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"tuck": app.Main,
	})
}

func setupScriptEnv(e *testscript.Env) error {
	oldUmask := syscall.Umask(0o022)
	e.Defer(func() {
		syscall.Umask(oldUmask)
	})

	e.Setenv("HOME", filepath.Join(e.WorkDir, "home"))
	e.Setenv("TUCK_TEST_STATE_DIR", filepath.Join(e.WorkDir, "state"))
	e.Setenv("XDG_STATE_HOME", filepath.Join(e.WorkDir, "state"))
	e.Setenv("XDG_CONFIG_HOME", filepath.Join(e.WorkDir, "xdg"))
	e.Setenv("NO_COLOR", "1")
	e.Setenv("TERM", "dumb")
	e.Setenv("LANG", "C")
	e.Setenv("LC_ALL", "C")
	e.Setenv("TUCK_TEST_ROOT_DIR", "")
	e.Setenv("TUCK_TEST_PRIVILEGE", "")

	return nil
}

func scriptCommands() map[string]func(ts *testscript.TestScript, neg bool, args []string) {
	return map[string]func(ts *testscript.TestScript, neg bool, args []string){
		"readlink": cmdReadlink,
		"wanthome": cmdWantHome,
	}
}

func cmdReadlink(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("readlink does not support !")
	}
	if len(args) != 2 {
		ts.Fatalf("usage: readlink <path> <expected-payload>")
	}

	got, err := os.Readlink(ts.MkAbs(args[0]))
	if err != nil {
		ts.Fatalf("readlink %s: %v", args[0], err)
	}
	if got != args[1] {
		ts.Fatalf("readlink %s: got payload %q, want %q", args[0], got, args[1])
	}
}

func cmdWantHome(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("wanthome does not support !")
	}
	if len(args) != 0 {
		ts.Fatalf("usage: wanthome")
	}

	home := ts.Getenv("HOME")
	stateRoot := ts.Getenv("TUCK_TEST_STATE_DIR")
	if home == "" || stateRoot == "" {
		ts.Fatalf("HOME and TUCK_TEST_STATE_DIR must be set by the harness")
	}

	if err := os.MkdirAll(home, 0o755); err != nil {
		ts.Fatalf("mkdir HOME: %v", err)
	}
	stateDir := filepath.Join(stateRoot, "tuck")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		ts.Fatalf("mkdir state dir: %v", err)
	}

	sources := fmt.Sprintf(`[[source]]
id = "public"
path = %q
enabled = true
default = true
`, filepath.Join(ts.Getenv("WORK"), "src"))
	if err := os.WriteFile(filepath.Join(stateDir, "sources.toml"), []byte(sources), 0o644); err != nil {
		ts.Fatalf("write sources.toml: %v", err)
	}
}
