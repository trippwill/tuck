package app

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunMetaJSONUsesSharedEnvelope(t *testing.T) {
	var out bytes.Buffer
	handled, err := runMetaJSON([]string{"tuck", "--version", "--json"}, &out)
	if err != nil {
		t.Fatalf("runMetaJSON() error = %v, want nil", err)
	}
	if !handled {
		t.Fatalf("runMetaJSON() handled = false, want true")
	}

	want := `{"schemaVersion":1,"command":"version","kind":"version","data":{"version":"dev"},"exitCode":0}` + "\n"
	if got := out.String(); got != want {
		t.Fatalf("runMetaJSON() output = %q, want %q", got, want)
	}
}

func TestRunMetaJSONSupportsPerCommandHelp(t *testing.T) {
	var out bytes.Buffer
	handled, err := runMetaJSON([]string{"tuck", "--json", "package", "use", "--help"}, &out)
	if err != nil {
		t.Fatalf("runMetaJSON() error = %v, want nil", err)
	}
	if !handled {
		t.Fatalf("runMetaJSON() handled = false, want true")
	}

	want := `{"schemaVersion":1,"command":"package use","kind":"help","data":{"name":"tuck package use","usage":"create managed symlinks for packages","argsUsage":"<package>...","flags":[{"name":"root","usage":"use the root context (target /); default is home"},{"name":"source","aliases":["s"],"usage":"select active source by enabled id"},{"name":"apply","usage":"execute the plan instead of just printing it"},{"name":"all","usage":"activate all packages in the active source"},{"name":"help","aliases":["h"],"usage":"show help"}],"globalFlags":[{"name":"json","usage":"machine-readable output"},{"name":"no-color","usage":"disable colored output (implied by --json)"}]},"exitCode":0}` + "\n"
	if got := out.String(); got != want {
		t.Fatalf("runMetaJSON() output = %q, want %q", got, want)
	}
}

func TestRunMetaJSONSupportsAliasHelp(t *testing.T) {
	var out bytes.Buffer
	handled, err := runMetaJSON([]string{"tuck", "pkg", "ls", "--help", "--json"}, &out)
	if err != nil {
		t.Fatalf("runMetaJSON() error = %v, want nil", err)
	}
	if !handled {
		t.Fatalf("runMetaJSON() handled = false, want true")
	}

	if got := out.String(); !strings.Contains(got, `"command":"package list"`) || !strings.Contains(got, `"aliases":["ls"]`) {
		t.Fatalf("runMetaJSON() output = %q, want canonical package list help with alias", got)
	}
}

func TestRunMetaJSONDoesNotHandleUnknownCommandHelp(t *testing.T) {
	var out bytes.Buffer
	handled, err := runMetaJSON([]string{"tuck", "--json", "missing", "--help"}, &out)
	if err != nil {
		t.Fatalf("runMetaJSON() error = %v, want nil", err)
	}
	if handled {
		t.Fatalf("runMetaJSON() handled = true, want false for unknown command")
	}
	if out.Len() != 0 {
		t.Fatalf("runMetaJSON() output = %q, want empty", out.String())
	}
}

func TestRunMetaJSONDoesNotHandleUnknownFlagHelp(t *testing.T) {
	var out bytes.Buffer
	handled, err := runMetaJSON([]string{"tuck", "--json", "--bogus", "--help"}, &out)
	if err != nil {
		t.Fatalf("runMetaJSON() error = %v, want nil", err)
	}
	if handled {
		t.Fatalf("runMetaJSON() handled = true, want false for unknown flag")
	}
	if out.Len() != 0 {
		t.Fatalf("runMetaJSON() output = %q, want empty", out.String())
	}
}

func TestRunMetaJSONDoesNotHandleOutOfScopeFlagHelp(t *testing.T) {
	var out bytes.Buffer
	handled, err := runMetaJSON([]string{"tuck", "--json", "--source", "public", "--help"}, &out)
	if err != nil {
		t.Fatalf("runMetaJSON() error = %v, want nil", err)
	}
	if handled {
		t.Fatalf("runMetaJSON() handled = true, want false for out-of-scope flag")
	}
	if out.Len() != 0 {
		t.Fatalf("runMetaJSON() output = %q, want empty", out.String())
	}
}
