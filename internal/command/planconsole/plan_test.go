package planconsole

import (
	"strings"
	"testing"

	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/plan"
)

func TestExitCodeFailsUnsatisfiedPrivilegeOnlyOnApply(t *testing.T) {
	satisfied := false
	dryRun := plan.Plan{
		DryRun:    true,
		Privilege: plan.Privilege{Required: true, Satisfied: &satisfied, Reason: "root-context write"},
	}
	if got := ExitCode(dryRun); got != output.ExitOK {
		t.Fatalf("dry-run ExitCode() = %d, want %d", got, output.ExitOK)
	}

	apply := dryRun
	apply.DryRun = false
	if got := ExitCode(apply); got != output.ExitFail {
		t.Fatalf("apply ExitCode() = %d, want %d", got, output.ExitFail)
	}
}

func TestConsoleStringReportsUnsatisfiedPrivilege(t *testing.T) {
	satisfied := false
	planned := plan.Plan{
		DryRun: false,
		Privilege: plan.Privilege{
			Required:  true,
			Satisfied: &satisfied,
			Reason:    "root-context write",
		},
		Actions: []plan.Action{plan.MkdirAction("/etc/ssh", "")},
	}

	got, err := ConsoleString(output.NewConsole(output.Invocation{Command: "command", Context: "root"}, false), planned)
	if err != nil {
		t.Fatalf("ConsoleString() error = %v", err)
	}
	if !strings.Contains(got, "privilege required: root-context write") {
		t.Fatalf("ConsoleString() = %q, want privilege requirement", got)
	}
}

func TestConsoleStringStylesConflictsWhenColorEnabled(t *testing.T) {
	planned := plan.Plan{
		DryRun: true,
		Conflicts: []plan.Conflict{{
			Code: "target_exists",
			Path: "~/.zshrc",
		}},
	}

	got, err := ConsoleString(output.NewConsole(output.Invocation{Command: "command", Context: "home"}, true), planned)
	if err != nil {
		t.Fatalf("ConsoleString() error = %v", err)
	}
	if !strings.Contains(got, "\x1b[31;1m!\x1b[0m \x1b[31;1mtarget_exists\x1b[0m ~/.zshrc") {
		t.Fatalf("ConsoleString() = %q, want styled conflict marker and code", got)
	}
}

func TestConsoleStringPrintsConflictHint(t *testing.T) {
	planned := plan.Plan{
		DryRun: true,
		Conflicts: []plan.Conflict{{
			Code: "real_file",
			Path: "~/.zshrc",
			Hint: "tuck adopt --replace zsh ~/.zshrc",
		}},
	}

	got, err := ConsoleString(output.NewConsole(output.Invocation{Command: "command", Context: "home"}, false), planned)
	if err != nil {
		t.Fatalf("ConsoleString() error = %v", err)
	}
	if !strings.Contains(got, "hint: tuck adopt --replace zsh ~/.zshrc") {
		t.Fatalf("ConsoleString() = %q, want conflict hint", got)
	}
}
