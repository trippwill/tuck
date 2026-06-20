package pkgcmd

import (
	"fmt"
	"io"

	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/packages"
)

const (
	CommandList  output.Command = "package list"
	KindPackages output.Kind    = "packages"
)

type ListRequest struct {
	SourceID string
	Context  string
}

type ListPayload struct {
	Source   string   `json:"source"`
	Packages []string `json:"packages"`
}

func List(req ListRequest) output.Outcome {
	listing, err := packages.List(packages.ListOptions{
		SourceID: req.SourceID,
		Context:  req.Context,
	})
	if err != nil {
		return output.Fail(err)
	}
	packageNames := make([]string, len(listing.Packages))
	copy(packageNames, listing.Packages)
	return output.OK(ListPayload{
		Source:   listing.Source,
		Packages: packageNames,
	})
}

func (p ListPayload) Kind() output.Kind {
	return KindPackages
}

func (p ListPayload) ExitCode() output.ExitCode {
	return output.ExitOK
}

func (p ListPayload) JSONData() any {
	return p
}

func (p ListPayload) WriteHuman(w io.Writer, inv output.Invocation) error {
	if _, err := fmt.Fprintf(w, "tuck %s   (context: %s, source: %s)\n\n", inv.Command, inv.Context, p.Source); err != nil {
		return err
	}
	for _, name := range p.Packages {
		if _, err := fmt.Fprintln(w, name); err != nil {
			return err
		}
	}
	if len(p.Packages) > 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "%d %s\n", len(p.Packages), packageNoun(len(p.Packages)))
	return err
}

func packageNoun(count int) string {
	if count == 1 {
		return "package"
	}
	return "packages"
}
