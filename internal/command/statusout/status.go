package statusout

import (
	"fmt"
	"strings"

	"github.com/trippwill/tuck/internal/output"
	statuspkg "github.com/trippwill/tuck/internal/status"
)

const Kind output.Kind = "status"

type Payload struct {
	Source  string  `json:"source"`
	Entries []Entry `json:"entries"`
}

type Entry struct {
	TargetPath     string `json:"targetPath"`
	State          string `json:"state"`
	Package        string `json:"package,omitempty"`
	Entry          string `json:"entry,omitempty"`
	Code           string `json:"code,omitempty"`
	Message        string `json:"message,omitempty"`
	Owner          string `json:"owner,omitempty"`
	ExpectedTarget string `json:"expectedTarget,omitempty"`
}

func FromResult(result statuspkg.Result) output.Result {
	payload := Payload{
		Source:  result.Source,
		Entries: fromEntries(result.Entries),
	}
	return output.Result{
		Kind:          Kind,
		Data:          payload,
		ExitCode:      output.ExitOK,
		ConsoleString: renderStatus,
	}
}

func renderStatus(inv output.Invocation, data any) (string, error) {
	p, ok := data.(Payload)
	if !ok {
		return "", fmt.Errorf("status console renderer received %T", data)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "tuck %s   (context: %s, source: %s)\n\n", inv.Command, inv.Context, p.Source)
	for _, entry := range p.Entries {
		fmt.Fprintf(&b, "%-14s %s", entry.State, entry.TargetPath)
		if entry.Package != "" {
			fmt.Fprintf(&b, " package=%s", entry.Package)
		}
		if entry.Entry != "" {
			fmt.Fprintf(&b, " entry=%s", entry.Entry)
		}
		if entry.Owner != "" && entry.Owner != entry.Package {
			fmt.Fprintf(&b, " owner=%s", entry.Owner)
		}
		if entry.Code != "" {
			fmt.Fprintf(&b, " code=%s", entry.Code)
		}
		if entry.Message != "" {
			fmt.Fprintf(&b, " (%s)", entry.Message)
		}
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "\n%d %s\n", len(p.Entries), entryNoun(len(p.Entries)))
	return b.String(), nil
}

func fromEntries(entries []statuspkg.Entry) []Entry {
	out := make([]Entry, len(entries))
	for i, entry := range entries {
		out[i] = Entry{
			TargetPath:     entry.TargetPath,
			State:          entry.State,
			Package:        entry.Package,
			Entry:          entry.Entry,
			Code:           string(entry.Code),
			Message:        entry.Message,
			Owner:          entry.Owner,
			ExpectedTarget: entry.ExpectedTarget,
		}
	}
	return out
}

func entryNoun(count int) string {
	if count == 1 {
		return "entry"
	}
	return "entries"
}
