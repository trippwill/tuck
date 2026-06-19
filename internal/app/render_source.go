package app

import (
	"fmt"

	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/state"
	"github.com/urfave/cli/v3"
)

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

func (r renderer) renderSourcesJSON(command string, registry state.Registry) error {
	return r.writeEnvelope(command, "", "sources", buildSourcesData(registry), ExitOK)
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

func renderSourceAdd(cmd *cli.Command, registry state.Registry, source state.Source) error {
	r := newRenderer(cmd)
	if r.json {
		return r.renderSourcesJSON("source add", registry)
	}

	defaultValue := "no"
	if registry.Default == source.ID {
		defaultValue = "yes"
	}
	fmt.Fprintf(r.out, "added source %s\n", source.ID)
	fmt.Fprintf(r.out, "path: %s\n", source.Path)
	fmt.Fprintf(r.out, "default: %s\n", defaultValue)
	return nil
}

func renderSourceRm(cmd *cli.Command, registry state.Registry, source state.Source) error {
	r := newRenderer(cmd)
	if r.json {
		return r.renderSourcesJSON("source rm", registry)
	}

	fmt.Fprintf(r.out, "removed source %s\n", source.ID)
	fmt.Fprintf(r.out, "path: %s\n", source.Path)
	return nil
}

func renderSourceDefault(cmd *cli.Command, registry state.Registry, source state.Source) error {
	r := newRenderer(cmd)
	if r.json {
		return r.renderSourcesJSON("source default", registry)
	}

	fmt.Fprintf(r.out, "default source %s\n", source.ID)
	fmt.Fprintf(r.out, "path: %s\n", source.Path)
	return nil
}

func renderSourceInit(cmd *cli.Command, initialized manifest.Initialized) error {
	r := newRenderer(cmd)
	if r.json {
		return r.renderSourcesJSON("source init", state.Registry{Sources: []state.Source{{
			ID:       initialized.Manifest.Name,
			Path:     initialized.Root,
			Enabled:  false,
			Manifest: initialized.Manifest,
		}}})
	}

	fmt.Fprintf(r.out, "initialized source %s\n", initialized.Manifest.Name)
	fmt.Fprintf(r.out, "path: %s\n", initialized.Path)
	if initialized.Manifest.Description != "" {
		fmt.Fprintf(r.out, "description: %s\n", initialized.Manifest.Description)
	}
	return nil
}

func renderSourceList(cmd *cli.Command, registry state.Registry) error {
	r := newRenderer(cmd)
	if r.json {
		return r.renderSourcesJSON("source list", registry)
	}

	if len(registry.Sources) == 0 {
		fmt.Fprintln(r.out, "no sources enabled")
		return nil
	}

	fmt.Fprintf(r.out, "%-8s %-8s %-8s %s\n", "ID", "DEFAULT", "ENABLED", "PATH")
	for _, source := range registry.Sources {
		defaultValue := "no"
		if registry.Default == source.ID {
			defaultValue = "yes"
		}
		enabledValue := "no"
		if source.Enabled {
			enabledValue = "yes"
		}
		fmt.Fprintf(r.out, "%-8s %-8s %-8s %s\n", source.ID, defaultValue, enabledValue, source.Path)
	}
	return nil
}
