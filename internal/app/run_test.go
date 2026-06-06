package app

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantExit       int
		wantStdoutPart string
		wantStderrPart string
	}{
		{
			name:           "help exits ok",
			args:           []string{"tuck", "--help"},
			wantExit:       ExitOK,
			wantStdoutPart: "usage",
		},
		{
			name:           "no command exits ok",
			args:           []string{"tuck"},
			wantExit:       ExitOK,
			wantStdoutPart: "usage",
		},
		{
			name:           "unknown command is usage",
			args:           []string{"tuck", "wat"},
			wantExit:       ExitUsage,
			wantStderrPart: "unknown command",
		},
		{
			name:           "unknown flag is usage",
			args:           []string{"tuck", "--bogus"},
			wantExit:       ExitUsage,
			wantStderrPart: "flag provided",
		},
		{
			name:           "global flags parse before help",
			args:           []string{"tuck", "--json", "--no-color", "--root", "--source", "public", "--help"},
			wantExit:       ExitOK,
			wantStdoutPart: "usage",
		},
		{
			name:           "version exits ok",
			args:           []string{"tuck", "--version"},
			wantExit:       ExitOK,
			wantStdoutPart: "tuck dev",
		},
		{
			name:           "version alias exits ok",
			args:           []string{"tuck", "-V"},
			wantExit:       ExitOK,
			wantStdoutPart: "tuck dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			got := Run(tt.args, nil, &stdout, &stderr)
			if got != tt.wantExit {
				t.Fatalf(`Run() exit = %d, want %d
stdout:
%s
stderr:
%s`, got, tt.wantExit, stdout.String(), stderr.String())
			}
			if tt.wantStdoutPart != "" && !strings.Contains(stdout.String(), tt.wantStdoutPart) {
				t.Fatalf("stdout = %q, want substring %q", stdout.String(), tt.wantStdoutPart)
			}
			if tt.wantStderrPart != "" && !strings.Contains(stderr.String(), tt.wantStderrPart) {
				t.Fatalf("stderr = %q, want substring %q", stderr.String(), tt.wantStderrPart)
			}
		})
	}
}
