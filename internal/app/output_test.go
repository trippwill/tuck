package app

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/trippwill/tuck/internal/output"
	"github.com/urfave/cli/v3"
)

func TestNoColorWinsOverTerminalDetection(t *testing.T) {
	t.Setenv("NO_COLOR", "")

	colored, _ := renderStyledProbe(t)
	if !strings.Contains(colored, "\x1b[32m") {
		t.Skip("/dev/ptmx is not terminal-like")
	}
	plain, stderr := renderStyledProbe(t, "--no-color")
	if strings.Contains(plain, "\x1b[") {
		t.Fatalf("human output with --no-color contains ANSI escapes: %q", plain)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	json, stderr := renderStyledProbe(t, "--json")
	if strings.Contains(json, "\x1b[") {
		t.Fatalf("JSON output contains ANSI escapes: %q", json)
	}
	if stderr != "" {
		t.Fatalf("JSON stderr = %q, want empty", stderr)
	}
}

func terminalLikeWriter(t *testing.T) *os.File {
	t.Helper()

	file, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("open terminal-like writer: %v", err)
	}
	return file
}

func renderStyledProbe(t *testing.T, args ...string) (string, string) {
	t.Helper()

	terminal := terminalLikeWriter(t)
	defer terminal.Close()
	stdout := &terminalBuffer{fd: terminal.Fd()}
	stderr := &terminalBuffer{fd: terminal.Fd()}
	cmd := &cli.Command{
		Name: "tuck",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json"},
			&cli.BoolFlag{Name: "no-color"},
		},
		Writer:    stdout,
		ErrWriter: stderr,
		Action: func(_ context.Context, cmd *cli.Command) error {
			_, err := rendererFor(cmd).Render(output.Invocation{Command: "probe"}, output.OK(output.Result{
				Kind:     "probe",
				Data:     map[string]string{"ok": "yes"},
				ExitCode: output.ExitOK,
				ConsoleString: func(console output.Console, _ any) (string, error) {
					return console.Style(output.StyleSuccess, "ok") + "\n", nil
				},
			}))
			return err
		},
	}
	if err := cmd.Run(context.Background(), append([]string{"tuck"}, args...)); err != nil {
		t.Fatalf("run styled probe: %v", err)
	}
	return stdout.String(), stderr.String()
}

type terminalBuffer struct {
	bytes.Buffer
	fd uintptr
}

func (b *terminalBuffer) Fd() uintptr {
	return b.fd
}
