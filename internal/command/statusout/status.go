package statusout

import (
	"fmt"
	"io"

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

func FromResult(result statuspkg.Result) Payload {
	return Payload{
		Source:  result.Source,
		Entries: fromEntries(result.Entries),
	}
}

func (p Payload) Kind() output.Kind {
	return Kind
}

func (p Payload) ExitCode() output.ExitCode {
	return output.ExitOK
}

func (p Payload) JSONData() any {
	return p
}

func (p Payload) WriteHuman(w io.Writer, inv output.Invocation) error {
	if _, err := fmt.Fprintf(w, "tuck %s   (context: %s, source: %s)\n\n", inv.Command, inv.Context, p.Source); err != nil {
		return err
	}
	for _, entry := range p.Entries {
		if _, err := fmt.Fprintf(w, "%-14s %s", entry.State, entry.TargetPath); err != nil {
			return err
		}
		if entry.Package != "" {
			if _, err := fmt.Fprintf(w, " package=%s", entry.Package); err != nil {
				return err
			}
		}
		if entry.Entry != "" {
			if _, err := fmt.Fprintf(w, " entry=%s", entry.Entry); err != nil {
				return err
			}
		}
		if entry.Owner != "" && entry.Owner != entry.Package {
			if _, err := fmt.Fprintf(w, " owner=%s", entry.Owner); err != nil {
				return err
			}
		}
		if entry.Code != "" {
			if _, err := fmt.Fprintf(w, " code=%s", entry.Code); err != nil {
				return err
			}
		}
		if entry.Message != "" {
			if _, err := fmt.Fprintf(w, " (%s)", entry.Message); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "\n%d %s\n", len(p.Entries), entryNoun(len(p.Entries)))
	return err
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
