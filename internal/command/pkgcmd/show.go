package pkgcmd

import (
	"fmt"
	"io"

	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/packages"
)

const (
	CommandShow output.Command = "package show"
	KindTree    output.Kind    = "tree"
)

type ShowRequest struct {
	SourceID string
	Context  string
	Ref      string
}

type TreePayload struct {
	Source  string      `json:"-"`
	Package TreePackage `json:"package"`
}

type TreePackage struct {
	Identity string      `json:"identity"`
	Root     string      `json:"root"`
	Entries  []TreeEntry `json:"entries"`
}

type TreeEntry struct {
	Rel  string `json:"rel"`
	Type string `json:"type"`
}

func Show(req ShowRequest) output.Outcome {
	tree, err := packages.Show(packages.ShowOptions{
		SourceID: req.SourceID,
		Context:  req.Context,
		Ref:      req.Ref,
	})
	if err != nil {
		return output.Fail(err)
	}
	return output.OK(TreePayload{
		Source: tree.Source,
		Package: TreePackage{
			Identity: tree.Package.Identity,
			Root:     tree.Package.Root,
			Entries:  fromTreeEntries(tree.Package.Entries),
		},
	})
}

func (p TreePayload) Kind() output.Kind {
	return KindTree
}

func (p TreePayload) ExitCode() output.ExitCode {
	return output.ExitOK
}

func (p TreePayload) JSONData() any {
	return p
}

func (p TreePayload) WriteHuman(w io.Writer, inv output.Invocation) error {
	if _, err := fmt.Fprintf(w, "tuck %s   (context: %s, source: %s)\n\n", inv.Command, inv.Context, p.Source); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "package: %s\n", p.Package.Identity); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "root: %s\n\n", p.Package.Root); err != nil {
		return err
	}
	for _, entry := range p.Package.Entries {
		if _, err := fmt.Fprintf(w, "%-4s %s\n", entry.Type, entry.Rel); err != nil {
			return err
		}
	}
	if len(p.Package.Entries) > 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "%d %s\n", len(p.Package.Entries), entryNoun(len(p.Package.Entries)))
	return err
}

func fromTreeEntries(entries []packages.TreeEntry) []TreeEntry {
	out := make([]TreeEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, TreeEntry{Rel: entry.Rel, Type: entry.Type})
	}
	return out
}

func entryNoun(count int) string {
	if count == 1 {
		return "entry"
	}
	return "entries"
}
