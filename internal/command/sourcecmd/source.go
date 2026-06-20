package sourcecmd

import (
	"fmt"
	"io"

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
		return output.Fail(output.InvalidArgs("--name requires --init", "add --init when writing a new source manifest"))
	}
	if !req.Init && req.Description != "" {
		return output.Fail(output.InvalidArgs("--description requires --init", "add --init when writing a new source manifest"))
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
		return output.Fail(err)
	}
	return output.OK(AddPayload{Registry: registry, Source: source})
}

func Init(req InitRequest) output.Outcome {
	initialized, err := manifest.Init(req.Path, manifest.InitOptions{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		return output.Fail(err)
	}
	return output.OK(InitPayload{Initialized: initialized})
}

func List() output.Outcome {
	registry, err := state.Load()
	if err != nil {
		return output.Fail(err)
	}
	return output.OK(ListPayload{Registry: registry})
}

func Rm(req IDRequest) output.Outcome {
	registry, removed, ok, err := state.RemoveSource(req.ID)
	if err != nil {
		return output.Fail(err)
	}
	if !ok {
		return output.Fail(resolve.ErrUnknownSource)
	}
	return output.OK(RmPayload{Registry: registry, Source: removed})
}

func Default(req IDRequest) output.Outcome {
	registry, source, ok, err := state.SetDefault(req.ID)
	if err != nil {
		return output.Fail(err)
	}
	if !ok {
		return output.Fail(resolve.ErrUnknownSource)
	}
	return output.OK(DefaultPayload{Registry: registry, Source: source})
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

func (p AddPayload) Kind() output.Kind     { return KindSources }
func (p InitPayload) Kind() output.Kind    { return KindSources }
func (p ListPayload) Kind() output.Kind    { return KindSources }
func (p RmPayload) Kind() output.Kind      { return KindSources }
func (p DefaultPayload) Kind() output.Kind { return KindSources }

func (p AddPayload) ExitCode() output.ExitCode     { return output.ExitOK }
func (p InitPayload) ExitCode() output.ExitCode    { return output.ExitOK }
func (p ListPayload) ExitCode() output.ExitCode    { return output.ExitOK }
func (p RmPayload) ExitCode() output.ExitCode      { return output.ExitOK }
func (p DefaultPayload) ExitCode() output.ExitCode { return output.ExitOK }

func (p AddPayload) JSONData() any     { return buildSourcesData(p.Registry) }
func (p ListPayload) JSONData() any    { return buildSourcesData(p.Registry) }
func (p RmPayload) JSONData() any      { return buildSourcesData(p.Registry) }
func (p DefaultPayload) JSONData() any { return buildSourcesData(p.Registry) }

func (p InitPayload) JSONData() any {
	return buildSourcesData(state.Registry{Sources: []state.Source{{
		ID:       p.Initialized.Manifest.Name,
		Path:     p.Initialized.Root,
		Enabled:  false,
		Manifest: p.Initialized.Manifest,
	}}})
}

func (p AddPayload) WriteHuman(w io.Writer, _ output.Invocation) error {
	defaultValue := "no"
	if p.Registry.Default == p.Source.ID {
		defaultValue = "yes"
	}
	if _, err := fmt.Fprintf(w, "added source %s\n", p.Source.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "path: %s\n", p.Source.Path); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "default: %s\n", defaultValue)
	return err
}

func (p InitPayload) WriteHuman(w io.Writer, _ output.Invocation) error {
	if _, err := fmt.Fprintf(w, "initialized source %s\n", p.Initialized.Manifest.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "path: %s\n", p.Initialized.Path); err != nil {
		return err
	}
	if p.Initialized.Manifest.Description != "" {
		if _, err := fmt.Fprintf(w, "description: %s\n", p.Initialized.Manifest.Description); err != nil {
			return err
		}
	}
	return nil
}

func (p ListPayload) WriteHuman(w io.Writer, _ output.Invocation) error {
	if len(p.Registry.Sources) == 0 {
		_, err := fmt.Fprintln(w, "no sources enabled")
		return err
	}
	if _, err := fmt.Fprintf(w, "%-8s %-8s %-8s %s\n", "ID", "DEFAULT", "ENABLED", "PATH"); err != nil {
		return err
	}
	for _, source := range p.Registry.Sources {
		defaultValue := "no"
		if p.Registry.Default == source.ID {
			defaultValue = "yes"
		}
		enabledValue := "no"
		if source.Enabled {
			enabledValue = "yes"
		}
		if _, err := fmt.Fprintf(w, "%-8s %-8s %-8s %s\n", source.ID, defaultValue, enabledValue, source.Path); err != nil {
			return err
		}
	}
	return nil
}

func (p RmPayload) WriteHuman(w io.Writer, _ output.Invocation) error {
	if _, err := fmt.Fprintf(w, "removed source %s\n", p.Source.ID); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "path: %s\n", p.Source.Path)
	return err
}

func (p DefaultPayload) WriteHuman(w io.Writer, _ output.Invocation) error {
	if _, err := fmt.Fprintf(w, "default source %s\n", p.Source.ID); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "path: %s\n", p.Source.Path)
	return err
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
