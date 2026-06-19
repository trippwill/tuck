package app

import (
	"io"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"
)

type versionData struct {
	Version string `json:"version"`
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
			flag, ok := resolveMetaFlag(root, current, arg)
			if !ok {
				return resolvedCommand{cmd: current, parts: parts, unknownFlag: true}
			}
			if flag.takesValue && !strings.Contains(arg, "=") {
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

type metaFlag struct {
	takesValue bool
}

func resolveMetaFlag(root *cli.Command, current *cli.Command, arg string) (metaFlag, bool) {
	name := flagName(arg)
	if flagMatches(cli.HelpFlag, name) {
		return metaFlag{}, true
	}
	if current == root && flagMatches(cli.VersionFlag, name) {
		return metaFlag{}, true
	}
	if flag, ok := findFlag(root.Flags, name); ok {
		return metaFlag{takesValue: flagTakesValue(flag)}, true
	}
	if flag, ok := findFlag(current.Flags, name); ok {
		return metaFlag{takesValue: flagTakesValue(flag)}, true
	}
	return metaFlag{}, false
}

func findFlag(flags []cli.Flag, name string) (cli.Flag, bool) {
	for _, flag := range flags {
		if flagMatches(flag, name) {
			return flag, true
		}
	}
	return nil, false
}

func flagMatches(flag cli.Flag, name string) bool {
	if flag == nil {
		return false
	}
	for _, flagName := range flag.Names() {
		if prefixedFlagName(flagName) == name {
			return true
		}
	}
	return false
}

func prefixedFlagName(name string) string {
	if len(name) == 1 {
		return "-" + name
	}
	return "--" + name
}

func flagTakesValue(flag cli.Flag) bool {
	docFlag, ok := flag.(cli.DocGenerationFlag)
	return ok && docFlag.TakesValue()
}

func flagName(arg string) string {
	if before, _, ok := strings.Cut(arg, "="); ok {
		return before
	}
	return arg
}
