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

func TestGeneratePackageLevelHelpers(t *testing.T) {
	dir := writePackage(t, map[string]string{
		"errors.go": `package sample

type ErrThing string

const (
	ErrMissing ErrThing = "missing thing"
	ErrInvalid ErrThing = "invalid thing"
)
`,
	})

	got, err := Generate(Options{Dir: dir, TypeName: "ErrThing"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	assertContains(t, string(got),
		"package sample",
		`import "github.com/trippwill/tuck/internal/apperr"`,
		"func (k ErrThing) Error() string",
		"type Error = apperr.Error[ErrThing]",
		"func AppErrMsg(sentinel ErrThing, msg string) error",
		"func AppErrMsgf(sentinel ErrThing, format string, args ...any) error",
		"func AppErrWrap(sentinel ErrThing, err error) error",
		"func AppErrWrapf(sentinel ErrThing, err error, format string, args ...any) error",
	)
	assertNotContains(t, string(got),
		"type SampleErr interface",
		"type Error[S ",
		"func AppErrMsg[S ",
		"func AppErr(",
		"func AppErrf(",
		"apperr.Wrap",
	)
}

func TestGenerateReportsConstraintUnsupported(t *testing.T) {
	dir := writePackage(t, map[string]string{
		"errors.go": `package sample

type ErrThing string

const ErrMissing ErrThing = "missing thing"
`,
	})

	_, err := Generate(Options{Dir: dir, TypeName: "ErrThing", ConstraintName: "ThingErr"})
	if err == nil {
		t.Fatalf("Generate() error = nil, want unsupported constraint error")
	}
	if !strings.Contains(err.Error(), `-constraint is not supported`) {
		t.Fatalf("Generate() error = %q, want unsupported constraint", err)
	}
}

func TestGenerateOutputFile(t *testing.T) {
	dir := writePackage(t, map[string]string{
		"errors.go": `package sample

type ErrThing string

const ErrMissing ErrThing = "missing thing"
`,
	})

	got, err := Generate(Options{Dir: dir, TypeName: "ErrThing", Output: "custom_gen.go"})
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

const (
	ErrMissing ErrThing = "missing thing"
	ErrInvalid ErrThing = "invalid thing"
)
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

	var appErr *Error
	if !errors.As(err, &appErr) {
		t.Fatalf("errors.As(err, *Error) = false, want true")
	}
	if got := appErr.Sentinel(); got != ErrInvalid {
		t.Fatalf("Sentinel() = %v, want %v", got, ErrInvalid)
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

	if _, err := Generate(Options{Dir: dir, TypeName: "ErrThing", Output: "apperr_gen.go"}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	cmd := exec.Command("go", "test", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("go test unexpectedly succeeded; want unselected sentinel compile failure")
	}
	if !strings.Contains(string(out), "errOther") || !strings.Contains(string(out), "as ErrThing") {
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

func TestGenerateReportsMissingTypeFlag(t *testing.T) {
	dir := writePackage(t, map[string]string{
		"errors.go": `package sample
`,
	})

	_, err := Generate(Options{Dir: dir})
	if err == nil {
		t.Fatalf("Generate() error = nil, want missing -type error")
	}
	if !strings.Contains(err.Error(), `-type is required`) {
		t.Fatalf("Generate() error = %q, want missing -type", err)
	}
}

func TestGenerateReportsCommaSeparatedType(t *testing.T) {
	dir := writePackage(t, map[string]string{
		"errors.go": `package sample

type ErrThing string
type ErrWidget string

const ErrMissing ErrThing = "missing thing"
const ErrBroken ErrWidget = "broken widget"
`,
	})

	_, err := Generate(Options{Dir: dir, TypeName: "ErrThing,ErrWidget"})
	if err == nil {
		t.Fatalf("Generate() error = nil, want comma-separated type error")
	}
	if !strings.Contains(err.Error(), `-type must name exactly one sentinel type`) {
		t.Fatalf("Generate() error = %q, want comma-separated type", err)
	}
}

func TestGenerateReportsMissingType(t *testing.T) {
	dir := writePackage(t, map[string]string{
		"errors.go": `package sample
`,
	})

	_, err := Generate(Options{Dir: dir, TypeName: "ErrThing"})
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

	_, err := Generate(Options{Dir: dir, TypeName: "ErrThing"})
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

	_, err := Generate(Options{Dir: dir, TypeName: "ErrThing"})
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
