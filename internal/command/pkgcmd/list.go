package pkgcmd

import (
	"fmt"
	"strings"

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
		return errorOutcome(err)
	}
	packageNames := make([]string, len(listing.Packages))
	copy(packageNames, listing.Packages)
	payload := ListPayload{
		Source:   listing.Source,
		Packages: packageNames,
	}
	return output.OK(output.Result{
		Kind:          KindPackages,
		Data:          payload,
		ExitCode:      output.ExitOK,
		ConsoleString: renderList,
	})
}

func renderList(inv output.Invocation, data any) (string, error) {
	p, ok := data.(ListPayload)
	if !ok {
		return "", fmt.Errorf("package list console renderer received %T", data)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "tuck %s   (context: %s, source: %s)\n\n", inv.Command, inv.Context, p.Source)
	for _, name := range p.Packages {
		fmt.Fprintln(&b, name)
	}
	if len(p.Packages) > 0 {
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "%d %s\n", len(p.Packages), packageNoun(len(p.Packages)))
	return b.String(), nil
}

func packageNoun(count int) string {
	if count == 1 {
		return "package"
	}
	return "packages"
}
