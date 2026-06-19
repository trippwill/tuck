package app

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"
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

type versionData struct {
	Version string `json:"version"`
}

type helpData struct {
	Name        string        `json:"name"`
	Usage       string        `json:"usage"`
	UsageText   string        `json:"usageText,omitempty"`
	ArgsUsage   string        `json:"argsUsage,omitempty"`
	Category    string        `json:"category,omitempty"`
	Aliases     []string      `json:"aliases,omitempty"`
	Commands    []commandData `json:"commands,omitempty"`
	Flags       []flagData    `json:"flags,omitempty"`
	GlobalFlags []flagData    `json:"globalFlags,omitempty"`
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
	if len(args) < 2 || !hasArg(args[1:], "--json") {
		return false, nil
	}
	root := rootCommand()
	resolved := resolveCommand(root, args[1:])
	if hasAnyArg(args[1:], "--version", "-v") && len(resolved.parts) == 0 && !resolved.unknown && !resolved.unknownFlag {
		return true, writeEnvelope(out, "version", "", "version", versionData{Version: version}, ExitOK)
	}
	if hasAnyArg(args[1:], "--help", "-h") {
		if resolved.unknown || resolved.unknownFlag {
			return false, nil
		}
		return true, writeEnvelope(out, resolved.commandName(), "", "help", buildHelpData(root, resolved), ExitOK)
	}
	return false, nil
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
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

type resolvedCommand struct {
	cmd         *cli.Command
	parts       []string
	unknown     bool
	unknownFlag bool
}

func (r resolvedCommand) commandName() string {
	if len(r.parts) == 0 {
		return "tuck"
	}
	return strings.Join(r.parts, " ")
}

func (r resolvedCommand) displayName() string {
	if len(r.parts) == 0 {
		return "tuck"
	}
	return "tuck " + strings.Join(r.parts, " ")
}

func resolveCommand(root *cli.Command, args []string) resolvedCommand {
	current := root
	parts := []string{}
	skipNext := false
	for _, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if arg == "--" {
			break
		}
		if strings.HasPrefix(arg, "-") {
			if !knownFlag(arg) || !flagAllowed(root, current, arg) {
				return resolvedCommand{cmd: current, parts: parts, unknownFlag: true}
			}
			if flagConsumesNext(arg) {
				skipNext = true
			}
			continue
		}
		child := findSubcommand(current, arg)
		if child == nil {
			if len(current.Commands) > 0 {
				return resolvedCommand{cmd: current, parts: parts, unknown: true}
			}
			break
		}
		current = child
		parts = append(parts, child.Name)
	}
	return resolvedCommand{cmd: current, parts: parts}
}

func findSubcommand(cmd *cli.Command, name string) *cli.Command {
	for _, child := range cmd.Commands {
		if child.Name == name || slices.Contains(child.Aliases, name) {
			return child
		}
	}
	return nil
}

func flagConsumesNext(arg string) bool {
	switch arg {
	case "--source", "-s", "--name", "--description":
		return true
	default:
		return false
	}
}

func knownFlag(arg string) bool {
	switch flagName(arg) {
	case "--json", "--no-color", "--help", "-h", "--version", "-v",
		"--source", "-s", "--root", "--apply", "--all",
		"--default", "--init", "--name", "--description":
		return true
	default:
		return false
	}
}

func flagAllowed(root *cli.Command, current *cli.Command, arg string) bool {
	name := flagName(arg)
	switch name {
	case "--json", "--no-color", "--help", "-h":
		return true
	case "--version", "-v":
		return current == root
	default:
		return commandHasFlag(current, name)
	}
}

func commandHasFlag(cmd *cli.Command, name string) bool {
	for _, flag := range cmd.Flags {
		record, ok := flagRecord(flag)
		if !ok {
			continue
		}
		if "--"+record.Name == name {
			return true
		}
		for _, alias := range record.Aliases {
			if "-"+alias == name {
				return true
			}
		}
	}
	return false
}

func flagName(arg string) string {
	if before, _, ok := strings.Cut(arg, "="); ok {
		return before
	}
	return arg
}

func buildHelpData(root *cli.Command, resolved resolvedCommand) helpData {
	cmd := resolved.cmd
	data := helpData{
		Name:      resolved.displayName(),
		Usage:     cmd.Usage,
		UsageText: cmd.UsageText,
		ArgsUsage: cmd.ArgsUsage,
		Category:  cmd.Category,
		Aliases:   append([]string(nil), cmd.Aliases...),
		Commands:  commandMetadata(cmd.Commands),
	}
	data.Flags = append(flagMetadata(cmd.Flags), flagData{Name: "help", Aliases: []string{"h"}, Usage: "show help"})
	if cmd == root {
		data.Flags = append(data.Flags, flagData{Name: "version", Aliases: []string{"v"}, Usage: "print the version"})
	} else {
		data.GlobalFlags = flagMetadata(root.Flags)
	}
	return data
}

func commandMetadata(commands []*cli.Command) []commandData {
	records := make([]commandData, 0, len(commands))
	for _, cmd := range commands {
		records = append(records, commandData{
			Name:    cmd.Name,
			Aliases: append([]string(nil), cmd.Aliases...),
			Usage:   cmd.Usage,
		})
	}
	return records
}

func flagMetadata(flags []cli.Flag) []flagData {
	records := make([]flagData, 0, len(flags))
	for _, flag := range flags {
		record, ok := flagRecord(flag)
		if ok {
			records = append(records, record)
		}
	}
	return records
}

func flagRecord(flag cli.Flag) (flagData, bool) {
	switch flag := flag.(type) {
	case *cli.BoolFlag:
		return flagData{Name: flag.Name, Aliases: append([]string(nil), flag.Aliases...), Usage: flag.Usage}, true
	case *cli.StringFlag:
		return flagData{Name: flag.Name, Aliases: append([]string(nil), flag.Aliases...), Usage: flag.Usage}, true
	default:
		return flagData{}, false
	}
}
