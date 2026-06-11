package app

import "github.com/urfave/cli/v3"

func requiredStringArgs(name, usage string) *cli.StringArgs {
	return &cli.StringArgs{
		Name:      name,
		UsageText: usage,
		Min:       1,
		Max:       1,
	}
}

func optionalStringArgs(name, usage string) *cli.StringArgs {
	return &cli.StringArgs{
		Name:      name,
		UsageText: usage,
		Max:       1,
	}
}

func variadicStringArgs(name, usage string, min int) *cli.StringArgs {
	return &cli.StringArgs{
		Name:      name,
		UsageText: usage,
		Min:       min,
		Max:       -1,
	}
}
