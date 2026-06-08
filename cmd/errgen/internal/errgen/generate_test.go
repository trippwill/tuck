package errgen

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGeneratePackageLevelConstrainedHelpers(t *testing.T) {
	dir := writePackage(t, map[string]string{
		"errors.go": `package sample

type ErrThing string
type ErrWidget string

const (
	ErrMissing ErrThing = "missing thing"
	ErrInvalid ErrThing = "invalid thing"
)

const ErrBroken ErrWidget = "broken widget"
`,
	})

	got, err := Generate(Options{Dir: dir, TypeNames: []string{"ErrThing", "ErrWidget"}})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	assertContains(t, string(got),
		"package sample",
		`import "github.com/trippwill/tuck/internal/apperr"`,
		"func (k ErrThing) Error() string",
		"func (k ErrWidget) Error() string",
		"type SampleErr interface",
		"error",
		"comparable",
		"ErrThing | ErrWidget",
		"type Error[S SampleErr] = apperr.Error[S]",
		"func AppErrMsg[S SampleErr](sentinel S, msg string) error",
		"func AppErrMsgf[S SampleErr](sentinel S, format string, args ...any) error",
		"func AppErrWrap[S SampleErr](sentinel S, err error) error",
		"func AppErrWrapf[S SampleErr](sentinel S, err error, format string, args ...any) error",
	)
	assertNotContains(t, string(got),
		"type Error = apperr.Error[",
		"func AppErr(",
		"func AppErrf(",
		"apperr.Wrap",
	)
}

func TestGenerateConstraintOverride(t *testing.T) {
	dir := writePackage(t, map[string]string{
		"errors.go": `package sample

type ErrThing string

const ErrMissing ErrThing = "missing thing"
`,
	})

	got, err := Generate(Options{Dir: dir, TypeNames: []string{"ErrThing"}, ConstraintName: "ThingErr"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	assertContains(t, string(got),
		"type ThingErr interface",
		"type Error[S ThingErr] = apperr.Error[S]",
		"func AppErrWrapf[S ThingErr](sentinel S, err error, format string, args ...any) error",
	)
	assertNotContains(t, string(got), "SampleErr")
}

func TestGenerateOutputFile(t *testing.T) {
	dir := writePackage(t, map[string]string{
		"errors.go": `package sample

type ErrThing string

const ErrMissing ErrThing = "missing thing"
`,
	})

	got, err := Generate(Options{Dir: dir, TypeNames: []string{"ErrThing"}, Output: "custom_gen.go"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("Generate() returned empty output")
	}
	if _, err := os.Stat(filepath.Join(dir, "custom_gen.go")); err != nil {
		t.Fatalf("custom output file was not written: %v", err)
	}
}

func TestGenerateCompilesAndPreservesErrorBehavior(t *testing.T) {
	dir := writePackage(t, map[string]string{
		"go.mod": `module github.com/trippwill/tuck/internal/errgenfixture

go 1.26

require github.com/trippwill/tuck v0.0.0

replace github.com/trippwill/tuck => ` + repoRoot(t) + `
`,
		"errors.go": `package errgenfixture

type ErrThing string
type ErrWidget string

const (
	ErrMissing ErrThing = "missing thing"
	ErrInvalid ErrThing = "invalid thing"
)

const ErrBroken ErrWidget = "broken widget"
`,
		"errors_test.go": `package errgenfixture

import (
	"errors"
	"testing"
)

type errOther string

func (e errOther) Error() string { return string(e) }

func TestGeneratedHelpers(t *testing.T) {
	cause := errors.New("disk failed")
	err := AppErrWrapf(ErrInvalid, cause, "could not read %s", "manifest")

	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("errors.Is(err, ErrInvalid) = false, want true")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(err, cause) = false, want true")
	}

	var appErr *Error[ErrThing]
	if !errors.As(err, &appErr) {
		t.Fatalf("errors.As(err, *Error[ErrThing]) = false, want true")
	}
	if got := appErr.Sentinel(); got != ErrInvalid {
		t.Fatalf("Sentinel() = %v, want %v", got, ErrInvalid)
	}

	widgetErr := AppErrMsg(ErrBroken, "bad widget")
	if !errors.Is(widgetErr, ErrBroken) {
		t.Fatalf("errors.Is(widgetErr, ErrBroken) = false, want true")
	}

	if got := AppErrMsg(ErrMissing, ""); got != ErrMissing {
		t.Fatalf("AppErrMsg(ErrMissing, empty) = %#v, want bare sentinel", got)
	}
}
`,
		"reject_test.go": `package errgenfixture

func compileRejectsUnselectedSentinel() {
	type errOther string
	const other errOther = "other"
	_ = AppErrMsg(other, "nope")
}
`,
	})

	if _, err := Generate(Options{Dir: dir, TypeNames: []string{"ErrThing", "ErrWidget"}, Output: "apperr_gen.go"}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	cmd := exec.Command("go", "test", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("go test unexpectedly succeeded; want unselected sentinel compile failure")
	}
	if !strings.Contains(string(out), "errOther") || !strings.Contains(string(out), "does not satisfy") {
		t.Fatalf("go test error did not reject unselected sentinel:\n%s", out)
	}

	if err := os.Remove(filepath.Join(dir, "reject_test.go")); err != nil {
		t.Fatalf("Remove reject_test.go error = %v", err)
	}
	cmd = exec.Command("go", "test", ".")
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test failed: %v\n%s", err, out)
	}
}

func TestGenerateReportsMissingTypes(t *testing.T) {
	dir := writePackage(t, map[string]string{
		"errors.go": `package sample
`,
	})

	_, err := Generate(Options{Dir: dir})
	if err == nil {
		t.Fatalf("Generate() error = nil, want missing -types error")
	}
	if !strings.Contains(err.Error(), `-types is required`) {
		t.Fatalf("Generate() error = %q, want missing -types", err)
	}
}

func TestGenerateReportsMissingType(t *testing.T) {
	dir := writePackage(t, map[string]string{
		"errors.go": `package sample
`,
	})

	_, err := Generate(Options{Dir: dir, TypeNames: []string{"ErrThing"}})
	if err == nil {
		t.Fatalf("Generate() error = nil, want missing type error")
	}
	if !strings.Contains(err.Error(), `type "ErrThing" not found`) {
		t.Fatalf("Generate() error = %q, want missing type", err)
	}
}

func TestGenerateReportsNonStringType(t *testing.T) {
	dir := writePackage(t, map[string]string{
		"errors.go": `package sample

type ErrThing int

const ErrMissing ErrThing = 1
`,
	})

	_, err := Generate(Options{Dir: dir, TypeNames: []string{"ErrThing"}})
	if err == nil {
		t.Fatalf("Generate() error = nil, want non-string type error")
	}
	if !strings.Contains(err.Error(), `type "ErrThing" must have underlying type string`) {
		t.Fatalf("Generate() error = %q, want non-string type", err)
	}
}

func TestGenerateReportsNoTypedConstants(t *testing.T) {
	dir := writePackage(t, map[string]string{
		"errors.go": `package sample

type ErrThing string
`,
	})

	_, err := Generate(Options{Dir: dir, TypeNames: []string{"ErrThing"}})
	if err == nil {
		t.Fatalf("Generate() error = nil, want no typed constants error")
	}
	if !strings.Contains(err.Error(), `no constants of type "ErrThing" found`) {
		t.Fatalf("Generate() error = %q, want no typed constants", err)
	}
}

func writePackage(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, contents := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}
	return dir
}

func assertContains(t *testing.T, haystack string, needles ...string) {
	t.Helper()

	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			t.Fatalf("generated source missing %q:\n%s", needle, haystack)
		}
	}
}

func assertNotContains(t *testing.T, haystack string, needles ...string) {
	t.Helper()

	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			t.Fatalf("generated source unexpectedly contains %q:\n%s", needle, haystack)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Fatalf("repo root %q does not contain go.mod", root)
		}
		t.Fatalf("stat repo go.mod: %v", err)
	}
	return root
}
