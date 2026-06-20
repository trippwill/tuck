package app

import (
	"context"

	"github.com/trippwill/tuck/internal/command/pkgcmd"
	"github.com/trippwill/tuck/internal/output"
	"github.com/urfave/cli/v3"
)

func packageCommand() *cli.Command {
	return &cli.Command{
		Name:            "package",
		Aliases:         []string{"pkg"},
		Usage:           "manage package symlinks",
		UsageText:       "tuck package <command> [args]",
		HideHelpCommand: true,
		CommandNotFound: commandNotFound,
		Commands: []*cli.Command{
			{
				Name:      "use",
				Usage:     "create managed symlinks for packages",
				ArgsUsage: "<package>...",
				Arguments: []cli.Argument{
					variadicStringArgs("package", "<package>...", 0),
				},
				OnUsageError: commandUsageError,
				Flags: append(
					mutatingFlags(),
					&cli.BoolFlag{Name: "all", Usage: "activate all packages in the active source"},
				),
				Action: packageUseAction,
			},
			{
				Name:      "drop",
				Usage:     "remove managed symlinks for packages",
				ArgsUsage: "<package>...",
				Arguments: []cli.Argument{
					variadicStringArgs("package", "<package>...", 1),
				},
				OnUsageError: commandUsageError,
				Flags:        mutatingFlags(),
				Action:       packageDropAction,
			},
			{
				Name:      "refresh",
				Usage:     "re-sync managed symlinks (drop + use)",
				ArgsUsage: "<package>...",
				Arguments: []cli.Argument{
					variadicStringArgs("package", "<package>...", 1),
				},
				OnUsageError: commandUsageError,
				Flags:        mutatingFlags(),
				Action:       packageRefreshAction,
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "list packages in the active source",
				Action:  packageListAction,
			},
			{
				Name:      "show",
				Aliases:   []string{"tree"},
				Usage:     "show package contents",
				ArgsUsage: "<package>",
				Arguments: []cli.Argument{
					requiredStringArgs("package", "<package>"),
				},
				OnUsageError: commandUsageError,
				Action:       packageShowAction,
			},
			{
				Name:      "status",
				Usage:     "show managed/conflict state for packages",
				ArgsUsage: "[package]",
				Arguments: []cli.Argument{
					optionalStringArgs("package", "[package]"),
				},
				OnUsageError: commandUsageError,
				Action:       packageStatusAction,
			},
			{
				Name:            "config",
				Usage:           "manage per-file package config",
				UsageText:       "tuck package config <command> [args]",
				HideHelpCommand: true,
				CommandNotFound: commandNotFound,
				Commands: []*cli.Command{
					{
						Name:      "show",
						Usage:     "show per-file package config",
						ArgsUsage: "<package> [path]",
						Arguments: []cli.Argument{
							&cli.StringArgs{
								Name:      "config",
								UsageText: "<package> [path]",
								Min:       1,
								Max:       2,
							},
						},
						OnUsageError: commandUsageError,
						Action:       packageConfigShowAction,
					},
					{
						Name:      "set",
						Usage:     "set per-file package config",
						ArgsUsage: "<package> <path>",
						Arguments: []cli.Argument{
							&cli.StringArgs{
								Name:      "config",
								UsageText: "<package> <path>",
								Min:       2,
								Max:       2,
							},
						},
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "deploy", Usage: "deployment strategy: symlink or copy"},
							&cli.StringFlag{Name: "mode", Usage: "octal mode for copied files"},
						},
						OnUsageError: commandUsageError,
						Action:       packageConfigSetAction,
					},
					{
						Name:      "unset",
						Usage:     "unset per-file package config",
						ArgsUsage: "<package> <path>",
						Arguments: []cli.Argument{
							&cli.StringArgs{
								Name:      "config",
								UsageText: "<package> <path>",
								Min:       2,
								Max:       2,
							},
						},
						Flags: []cli.Flag{
							&cli.BoolFlag{Name: "deploy", Usage: "unset deploy for the file"},
							&cli.BoolFlag{Name: "mode", Usage: "unset mode for the file"},
						},
						OnUsageError: commandUsageError,
						Action:       packageConfigUnsetAction,
					},
				},
			},
		},
	}
}

func packageUseAction(_ context.Context, cmd *cli.Command) error {
	contextName := contextFromFlag(cmd)
	outcome := pkgcmd.Use(pkgcmd.UseRequest{
		Refs:     cmd.StringArgs("package"),
		All:      cmd.Bool("all"),
		SourceID: cmd.String("source"),
		Context:  contextName,
		Apply:    cmd.Bool("apply"),
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: pkgcmd.CommandUse,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}

func packageDropAction(_ context.Context, cmd *cli.Command) error {
	contextName := contextFromFlag(cmd)
	outcome := pkgcmd.Drop(pkgcmd.DropRequest{
		Refs:     cmd.StringArgs("package"),
		SourceID: cmd.String("source"),
		Context:  contextName,
		Apply:    cmd.Bool("apply"),
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: pkgcmd.CommandDrop,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}

func packageRefreshAction(_ context.Context, cmd *cli.Command) error {
	contextName := contextFromFlag(cmd)
	outcome := pkgcmd.Refresh(pkgcmd.RefreshRequest{
		Refs:     cmd.StringArgs("package"),
		SourceID: cmd.String("source"),
		Context:  contextName,
		Apply:    cmd.Bool("apply"),
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: pkgcmd.CommandRefresh,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}

func packageListAction(_ context.Context, cmd *cli.Command) error {
	if err := noExtraArgs(cmd, "package list", "package list accepts no arguments", "remove the extra argument and retry"); err != nil {
		return err
	}
	contextName := contextFromFlag(cmd)
	outcome := pkgcmd.List(pkgcmd.ListRequest{
		SourceID: cmd.String("source"),
		Context:  contextName,
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: pkgcmd.CommandList,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}

func packageStatusAction(_ context.Context, cmd *cli.Command) error {
	if err := noExtraArgs(cmd, "package status", "package status accepts at most one package ref", "remove the extra package ref and retry"); err != nil {
		return err
	}
	ref := ""
	if refs := cmd.StringArgs("package"); len(refs) > 0 {
		ref = refs[0]
	}
	contextName := contextFromFlag(cmd)
	outcome := pkgcmd.Status(pkgcmd.StatusRequest{
		Ref:      ref,
		SourceID: cmd.String("source"),
		Context:  contextName,
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: pkgcmd.CommandStatus,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}

func packageShowAction(_ context.Context, cmd *cli.Command) error {
	if err := noExtraArgs(cmd, "package show", "package show accepts exactly one package ref", "pass exactly one package name"); err != nil {
		return err
	}
	contextName := contextFromFlag(cmd)
	outcome := pkgcmd.Show(pkgcmd.ShowRequest{
		SourceID: cmd.String("source"),
		Context:  contextName,
		Ref:      cmd.StringArgs("package")[0],
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: pkgcmd.CommandShow,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}

func packageConfigShowAction(_ context.Context, cmd *cli.Command) error {
	args := cmd.StringArgs("config")
	path := ""
	if len(args) > 1 {
		path = args[1]
	}
	contextName := contextFromFlag(cmd)
	outcome := pkgcmd.ConfigShow(pkgcmd.ConfigShowRequest{
		Ref:      args[0],
		Path:     path,
		SourceID: cmd.String("source"),
		Context:  contextName,
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: pkgcmd.CommandConfigShow,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}

func packageConfigSetAction(_ context.Context, cmd *cli.Command) error {
	args := cmd.StringArgs("config")
	contextName := contextFromFlag(cmd)
	outcome := pkgcmd.ConfigSet(pkgcmd.ConfigSetRequest{
		Ref:       args[0],
		Path:      args[1],
		Deploy:    cmd.String("deploy"),
		Mode:      cmd.String("mode"),
		SetDeploy: cmd.IsSet("deploy"),
		SetMode:   cmd.IsSet("mode"),
		SourceID:  cmd.String("source"),
		Context:   contextName,
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: pkgcmd.CommandConfigSet,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}

func packageConfigUnsetAction(_ context.Context, cmd *cli.Command) error {
	args := cmd.StringArgs("config")
	contextName := contextFromFlag(cmd)
	outcome := pkgcmd.ConfigUnset(pkgcmd.ConfigUnsetRequest{
		Ref:         args[0],
		Path:        args[1],
		UnsetDeploy: cmd.Bool("deploy"),
		UnsetMode:   cmd.Bool("mode"),
		SourceID:    cmd.String("source"),
		Context:     contextName,
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: pkgcmd.CommandConfigUnset,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}
