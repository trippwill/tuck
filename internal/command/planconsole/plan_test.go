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

	got, err := ConsoleString(output.Invocation{Command: "command", Context: "root"}, planned)
	if err != nil {
		t.Fatalf("ConsoleString() error = %v", err)
	}
	if !strings.Contains(got, "privilege required: root-context write") {
		t.Fatalf("ConsoleString() = %q, want privilege requirement", got)
	}
}
