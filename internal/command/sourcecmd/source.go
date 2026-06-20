package sourcecmd

import (
	"fmt"
	"strings"

	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/resolve"
	"github.com/trippwill/tuck/internal/state"
)

const (
	CommandAdd     output.Command = "source add"
	CommandInit    output.Command = "source init"
	CommandRm      output.Command = "source rm"
	CommandList    output.Command = "source list"
	CommandDefault output.Command = "source default"
	KindSources    output.Kind    = "sources"
)

type AddRequest struct {
	Path        string
	Default     bool
	Init        bool
	Name        string
	Description string
}

type InitRequest struct {
	Path        string
	Name        string
	Description string
}

type IDRequest struct {
	ID string
}

func Add(req AddRequest) output.Outcome {
	if !req.Init && req.Name != "" {
		return output.OK(output.InvalidArgs("--name requires --init", "add --init when writing a new source manifest"))
	}
	if !req.Init && req.Description != "" {
		return output.OK(output.InvalidArgs("--description requires --init", "add --init when writing a new source manifest"))
	}

	var (
		registry state.Registry
		source   state.Source
		err      error
	)
	if req.Init {
		registry, source, err = state.AddSourceWithInit(req.Path, req.Default, manifest.InitOptions{
			Name:        req.Name,
			Description: req.Description,
		})
	} else {
		registry, source, err = state.AddSource(req.Path, req.Default)
	}
	if err != nil {
		return errorOutcome(err)
	}
	return output.OK(addResult(AddPayload{Registry: registry, Source: source}))
}

func Init(req InitRequest) output.Outcome {
	initialized, err := manifest.Init(req.Path, manifest.InitOptions{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		return errorOutcome(err)
	}
	return output.OK(initResult(InitPayload{Initialized: initialized}))
}

func List() output.Outcome {
	registry, err := state.Load()
	if err != nil {
		return errorOutcome(err)
	}
	return output.OK(listResult(ListPayload{Registry: registry}))
}

func Rm(req IDRequest) output.Outcome {
	registry, removed, ok, err := state.RemoveSource(req.ID)
	if err != nil {
		return errorOutcome(err)
	}
	if !ok {
		return errorOutcome(resolve.ErrUnknownSource)
	}
	return output.OK(rmResult(RmPayload{Registry: registry, Source: removed}))
}

func Default(req IDRequest) output.Outcome {
	registry, source, ok, err := state.SetDefault(req.ID)
	if err != nil {
		return errorOutcome(err)
	}
	if !ok {
		return errorOutcome(resolve.ErrUnknownSource)
	}
	return output.OK(defaultResult(DefaultPayload{Registry: registry, Source: source}))
}

type AddPayload struct {
	Registry state.Registry
	Source   state.Source
}

type InitPayload struct {
	Initialized manifest.Initialized
}

type ListPayload struct {
	Registry state.Registry
}

type RmPayload struct {
	Registry state.Registry
	Source   state.Source
}

type DefaultPayload struct {
	Registry state.Registry
	Source   state.Source
}

func addResult(p AddPayload) output.Result {
	return output.Result{
		Kind:          KindSources,
		Data:          buildSourcesData(p.Registry),
		ExitCode:      output.ExitOK,
		ConsoleString: func(output.Invocation, any) (string, error) { return renderAdd(p), nil },
	}
}

func initResult(p InitPayload) output.Result {
	return output.Result{
		Kind:          KindSources,
		Data:          initData(p),
		ExitCode:      output.ExitOK,
		ConsoleString: func(output.Invocation, any) (string, error) { return renderInit(p), nil },
	}
}

func listResult(p ListPayload) output.Result {
	return output.Result{
		Kind:          KindSources,
		Data:          buildSourcesData(p.Registry),
		ExitCode:      output.ExitOK,
		ConsoleString: func(output.Invocation, any) (string, error) { return renderList(p), nil },
	}
}

func rmResult(p RmPayload) output.Result {
	return output.Result{
		Kind:          KindSources,
		Data:          buildSourcesData(p.Registry),
		ExitCode:      output.ExitOK,
		ConsoleString: func(output.Invocation, any) (string, error) { return renderRm(p), nil },
	}
}

func defaultResult(p DefaultPayload) output.Result {
	return output.Result{
		Kind:          KindSources,
		Data:          buildSourcesData(p.Registry),
		ExitCode:      output.ExitOK,
		ConsoleString: func(output.Invocation, any) (string, error) { return renderDefault(p), nil },
	}
}

func initData(p InitPayload) sourcesData {
	return buildSourcesData(state.Registry{Sources: []state.Source{{
		ID:       p.Initialized.Manifest.Name,
		Path:     p.Initialized.Root,
		Enabled:  false,
		Manifest: p.Initialized.Manifest,
	}}})
}

func renderAdd(p AddPayload) string {
	defaultValue := "no"
	if p.Registry.Default == p.Source.ID {
		defaultValue = "yes"
	}
	return fmt.Sprintf("added source %s\npath: %s\ndefault: %s\n", p.Source.ID, p.Source.Path, defaultValue)
}

func renderInit(p InitPayload) string {
	var b strings.Builder
	fmt.Fprintf(&b, "initialized source %s\n", p.Initialized.Manifest.Name)
	fmt.Fprintf(&b, "path: %s\n", p.Initialized.Path)
	if p.Initialized.Manifest.Description != "" {
		fmt.Fprintf(&b, "description: %s\n", p.Initialized.Manifest.Description)
	}
	return b.String()
}

func renderList(p ListPayload) string {
	if len(p.Registry.Sources) == 0 {
		return "no sources enabled\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%-8s %-8s %-8s %s\n", "ID", "DEFAULT", "ENABLED", "PATH")
	for _, source := range p.Registry.Sources {
		defaultValue := "no"
		if p.Registry.Default == source.ID {
			defaultValue = "yes"
		}
		enabledValue := "no"
		if source.Enabled {
			enabledValue = "yes"
		}
		fmt.Fprintf(&b, "%-8s %-8s %-8s %s\n", source.ID, defaultValue, enabledValue, source.Path)
	}
	return b.String()
}

func renderRm(p RmPayload) string {
	return fmt.Sprintf("removed source %s\npath: %s\n", p.Source.ID, p.Source.Path)
}

func renderDefault(p DefaultPayload) string {
	return fmt.Sprintf("default source %s\npath: %s\n", p.Source.ID, p.Source.Path)
}

type sourcesData struct {
	Sources []sourceRecord `json:"sources"`
}

type sourceRecord struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	Enabled     bool   `json:"enabled"`
	Default     bool   `json:"default"`
	Description string `json:"description"`
}

func buildSourcesData(registry state.Registry) sourcesData {
	records := make([]sourceRecord, 0, len(registry.Sources))
	for _, source := range registry.Sources {
		records = append(records, sourceRecord{
			ID:          source.ID,
			Path:        source.Path,
			Enabled:     source.Enabled,
			Default:     registry.Default == source.ID,
			Description: source.Manifest.Description,
		})
	}
	return sourcesData{Sources: records}
}
