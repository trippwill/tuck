package app

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"slices"
)

func Main() {
	if handled, err := runMetaJSON(os.Args, os.Stdout); handled {
		if err != nil {
			os.Exit(ExitFail)
		}
		return
	}
	if err := rootCommand().Run(context.Background(), os.Args); err != nil {
		os.Exit(ExitFail)
	}
}

type metaEnvelope struct {
	SchemaVersion int    `json:"schemaVersion"`
	Command       string `json:"command"`
	Kind          string `json:"kind"`
	Data          any    `json:"data"`
	ExitCode      int    `json:"exitCode"`
}

type versionData struct {
	Version string `json:"version"`
}

type helpData struct {
	Name     string        `json:"name"`
	Usage    string        `json:"usage"`
	Commands []commandData `json:"commands"`
	Flags    []flagData    `json:"flags"`
}

type commandData struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases,omitempty"`
	Usage   string   `json:"usage"`
}

type flagData struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases,omitempty"`
	Usage   string   `json:"usage"`
}

func runMetaJSON(args []string, out io.Writer) (bool, error) {
	if !hasArg(args[1:], "--json") {
		return false, nil
	}
	if hasAnyArg(args[1:], "--version", "-v") && !hasNonFlag(args[1:]) {
		return true, writeJSON(out, metaEnvelope{
			SchemaVersion: 1,
			Command:       "version",
			Kind:          "version",
			Data:          versionData{Version: version},
			ExitCode:      ExitOK,
		})
	}
	if hasAnyArg(args[1:], "--help", "-h") && !hasNonFlag(args[1:]) {
		return true, writeJSON(out, metaEnvelope{
			SchemaVersion: 1,
			Command:       "tuck",
			Kind:          "help",
			Data: helpData{
				Name:  "tuck",
				Usage: "manage dotfiles by linking package leaves into a target tree",
				Commands: []commandData{
					{Name: "adopt", Usage: "move a real file into a package, then link it back"},
					{Name: "eject", Usage: "remove a managed link, restoring the real file"},
					{Name: "status", Usage: "classify a target path (managed/conflict/absent)"},
					{Name: "package", Aliases: []string{"pkg"}, Usage: "manage package symlinks"},
					{Name: "source", Usage: "manage enabled dotfiles sources"},
				},
				Flags: []flagData{
					{Name: "json", Usage: "machine-readable output"},
					{Name: "no-color", Usage: "disable colored output (implied by --json)"},
					{Name: "help", Aliases: []string{"h"}, Usage: "show help"},
					{Name: "version", Aliases: []string{"v"}, Usage: "print the version"},
				},
			},
			ExitCode: ExitOK,
		})
	}
	return false, nil
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	return encoder.Encode(value)
}

func hasAnyArg(args []string, wants ...string) bool {
	for _, want := range wants {
		if hasArg(args, want) {
			return true
		}
	}
	return false
}

func hasArg(args []string, want string) bool {
	return slices.Contains(args, want)
}

func hasNonFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if len(arg) > 0 && arg[0] == '-' {
			continue
		}
		return true
	}
	return false
}
