package app

import "github.com/urfave/cli/v3"

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
	data.Flags = append(flagMetadata(cmd.Flags), mustFlagRecord(cli.HelpFlag))
	if cmd == root {
		data.Flags = append(data.Flags, mustFlagRecord(cli.VersionFlag))
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

func mustFlagRecord(flag cli.Flag) flagData {
	record, ok := flagRecord(flag)
	if !ok {
		return flagData{}
	}
	return record
}

func flagRecord(flag cli.Flag) (flagData, bool) {
	if flag == nil {
		return flagData{}, false
	}
	names := flag.Names()
	if len(names) == 0 {
		return flagData{}, false
	}
	record := flagData{
		Name:    names[0],
		Aliases: append([]string(nil), names[1:]...),
	}
	if docFlag, ok := flag.(cli.DocGenerationFlag); ok {
		record.Usage = docFlag.GetUsage()
	}
	return record, true
}
