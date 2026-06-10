//go:build tuck_testhooks

package acceptance

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
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

func runScriptSuite(t *testing.T, suite string) {
	t.Helper()

	testscript.Run(t, testscript.Params{
		Dir:   filepath.Join("testdata", "script", suite),
		Setup: setupScriptEnv,
		Cmds:  scriptCommands(),
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
		"phase":     cmdPhase,
		"readlink":  cmdReadlink,
		"wantstate": cmdWantState,
		"wanthome":  cmdWantHome,
	}
}

func cmdPhase(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("phase does not support !")
	}
	if len(args) < 1 {
		ts.Fatalf("usage: phase <name> [source-fixture]...")
	}
	name := filepath.Clean(args[0])
	if name == "." || filepath.IsAbs(name) || name == ".." || filepath.Dir(name) != "." {
		ts.Fatalf("phase name %q must be a single relative path segment", args[0])
	}

	home := filepath.Join(ts.Getenv("WORK"), "home")
	stateRoot := filepath.Join(ts.Getenv("WORK"), "state", name)
	configRoot := filepath.Join(ts.Getenv("WORK"), "xdg", name)
	sourceRoot := filepath.Join(ts.Getenv("WORK"), "src")
	for _, path := range []string{home, stateRoot, configRoot, sourceRoot} {
		if err := os.RemoveAll(path); err != nil {
			ts.Fatalf("reset phase path %s: %v", path, err)
		}
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		ts.Fatalf("mkdir phase HOME: %v", err)
	}
	for _, fixture := range args[1:] {
		if err := copyTree(ts.MkAbs(fixture), sourceRoot); err != nil {
			ts.Fatalf("copy source fixture: %v", err)
		}
	}

	ts.Setenv("HOME", home)
	ts.Setenv("TUCK_TEST_STATE_DIR", stateRoot)
	ts.Setenv("XDG_STATE_HOME", stateRoot)
	ts.Setenv("XDG_CONFIG_HOME", configRoot)
	ts.Setenv("TUCK_TEST_ROOT_DIR", "")
	ts.Setenv("TUCK_TEST_PRIVILEGE", "")
}

func copyTree(src string, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}
	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == src {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyFile(path, target)
	})
}

func copyFile(src string, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
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

	sourcePath := filepath.Join(ts.Getenv("WORK"), "src")
	if err := os.MkdirAll(sourcePath, 0o755); err != nil {
		ts.Fatalf("mkdir source repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourcePath, "tuck.toml"), []byte("name = \"public\"\n"), 0o644); err != nil {
		ts.Fatalf("write source manifest: %v", err)
	}

	sources := fmt.Sprintf(`default = "public"

[[source]]
id = "public"
path = %q
enabled = true
`, sourcePath)
	if err := os.WriteFile(filepath.Join(stateDir, "sources.toml"), []byte(sources), 0o644); err != nil {
		ts.Fatalf("write sources.toml: %v", err)
	}
}

func cmdWantState(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("wantstate does not support !")
	}
	if len(args) < 1 || (len(args)-1)%3 != 0 {
		ts.Fatalf("usage: wantstate <default-id|-> [<id> <path> <enabled>]...")
	}

	stateRoot := ts.Getenv("TUCK_TEST_STATE_DIR")
	if stateRoot == "" {
		ts.Fatalf("TUCK_TEST_STATE_DIR must be set by the harness")
	}
	stateDir := filepath.Join(stateRoot, "tuck")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		ts.Fatalf("mkdir state dir: %v", err)
	}

	defaultID := args[0]
	sources := ""
	if defaultID != "-" {
		sources += fmt.Sprintf("default = %q\n\n", defaultID)
	}
	for i := 1; i < len(args); i += 3 {
		enabled, err := strconv.ParseBool(args[i+2])
		if err != nil {
			ts.Fatalf("parse enabled value %q: %v", args[i+2], err)
		}
		sourcePath := args[i+1]
		if !filepath.IsAbs(sourcePath) {
			sourcePath = ts.MkAbs(sourcePath)
		}
		if i > 1 {
			sources += "\n"
		}
		sources += fmt.Sprintf("[[source]]\nid = %q\npath = %q\nenabled = %t\n", args[i], filepath.Clean(sourcePath), enabled)
	}

	if err := os.WriteFile(filepath.Join(stateDir, "sources.toml"), []byte(sources), 0o644); err != nil {
		ts.Fatalf("write sources.toml: %v", err)
	}
}
