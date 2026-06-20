package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRendererDoesNotColorJSON(t *testing.T) {
	var out bytes.Buffer
	var stderr bytes.Buffer
	renderer := NewRenderer(Options{
		Format:   JSON,
		Color:    true,
		ErrColor: true,
		Out:      &out,
		Err:      &stderr,
	})

	exitCode, err := renderer.Render(Invocation{Command: "command"}, OK(ErrorResult("bad", "failed", "fix it")))
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if exitCode != ExitFail {
		t.Fatalf("Render() exitCode = %d, want %d", exitCode, ExitFail)
	}
	if strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("Render() JSON output contains ANSI escapes: %q", out.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("Render() stderr = %q, want empty", stderr.String())
	}
}

func TestRendererColorsErrorStreamWithErrColor(t *testing.T) {
	var out bytes.Buffer
	var stderr bytes.Buffer
	renderer := NewRenderer(Options{
		Format:   Human,
		Color:    false,
		ErrColor: true,
		Out:      &out,
		Err:      &stderr,
	})

	exitCode, err := renderer.Render(Invocation{Command: "command"}, OK(ErrorResult("bad", "failed", "fix it")))
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if exitCode != ExitFail {
		t.Fatalf("Render() exitCode = %d, want %d", exitCode, ExitFail)
	}
	if out.Len() != 0 {
		t.Fatalf("Render() stdout = %q, want empty", out.String())
	}
	if got := stderr.String(); !strings.Contains(got, "\x1b[31;1merror:\x1b[0m failed") {
		t.Fatalf("Render() stderr = %q, want styled error label", got)
	}
}

func TestRendererColorsResultStreamWithColor(t *testing.T) {
	var out bytes.Buffer
	var stderr bytes.Buffer
	renderer := NewRenderer(Options{
		Format: Human,
		Color:  true,
		Out:    &out,
		Err:    &stderr,
	})

	exitCode, err := renderer.Render(Invocation{Command: "command"}, OK(Result{
		Kind:     "test",
		ExitCode: ExitOK,
		ConsoleString: func(console Console, _ any) (string, error) {
			return console.Style(StyleSuccess, "ok") + "\n", nil
		},
	}))
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if exitCode != ExitOK {
		t.Fatalf("Render() exitCode = %d, want %d", exitCode, ExitOK)
	}
	if got := out.String(); got != "\x1b[32mok\x1b[0m\n" {
		t.Fatalf("Render() stdout = %q, want styled result", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("Render() stderr = %q, want empty", stderr.String())
	}
}
