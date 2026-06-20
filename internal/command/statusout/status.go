package statusout

import (
	"fmt"
	"strings"

	"github.com/trippwill/tuck/internal/output"
	statuspkg "github.com/trippwill/tuck/internal/status"
)

const Kind output.Kind = "status"

func FromResult(result statuspkg.Result) output.Result {
	return output.Result{
		Kind:          Kind,
		Data:          result,
		ExitCode:      output.ExitOK,
		ConsoleString: renderStatus,
	}
}

func renderStatus(console output.Console, data any) (string, error) {
	p, ok := data.(statuspkg.Result)
	if !ok {
		return "", fmt.Errorf("status console renderer received %T", data)
	}
	inv := console.Invocation
	var b strings.Builder
	fmt.Fprintf(&b, "tuck %s   (context: %s, source: %s)\n\n", inv.Command, inv.Context, p.Source)
	for _, entry := range p.Entries {
		fmt.Fprintf(&b, "%s %s", console.Style(statusStyle(entry.State), fmt.Sprintf("%-14s", entry.State)), entry.TargetPath)
		if entry.Package != "" {
			fmt.Fprintf(&b, " package=%s", console.Style(output.StylePackage, entry.Package))
		}
		if entry.Entry != "" {
			fmt.Fprintf(&b, " entry=%s", entry.Entry)
		}
		if entry.Owner != "" && entry.Owner != entry.Package {
			fmt.Fprintf(&b, " owner=%s", console.Style(output.StylePackage, entry.Owner))
		}
		if entry.Code != "" {
			fmt.Fprintf(&b, " code=%s", console.Style(output.StyleDanger, string(entry.Code)))
		}
		if entry.Message != "" {
			fmt.Fprintf(&b, " (%s)", entry.Message)
		}
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "\n%d %s\n", len(p.Entries), entryNoun(len(p.Entries)))
	return b.String(), nil
}

func statusStyle(state string) output.Style {
	switch state {
	case statuspkg.StateDeployed:
		return output.StyleSuccess
	case statuspkg.StateAbsent:
		return output.StyleWarning
	case statuspkg.StateConflict, statuspkg.StateMismatch, statuspkg.StateOwnedByOther:
		return output.StyleDanger
	case statuspkg.StateUnmanaged:
		return output.StyleWarning
	default:
		return output.StyleMuted
	}
}

func entryNoun(count int) string {
	if count == 1 {
		return "entry"
	}
	return "entries"
}
