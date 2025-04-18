// Copyright (c) 2025 Visvasity LLC

// Package cli provides a lightweight framework for creating command-line
// interfaces (CLIs). It supports defining commands as functions or objects,
// organizing them into subcommand groups, parsing flags using the
// [flag.FlagSet]s, and generating documentation via built-in commands: "help",
// "flags", and "commands".
//
// Key features:
//   - Commands defined as functions or objects implementing the Command interface.
//   - Hierarchical subcommand groups.
//   - Flag parsing using flag.FlagSet with custom error handling.
//   - Automatic documentation through built-in commands.
//   - Custom documentation using optional interfaces.
//   - Context-aware execution for cancellation and timeouts.
//
// Create commands with [NewCommand] for functions, [NewGroup] for subcommands,
// or custom types implementing the [Command] interface. Execute the CLI by passing
// commands to [Run] with command-line arguments.
//
// Example (Function-based command):
//
//	var listFlags flag.FlagSet
//	verbose := listFlags.Bool("v", false, "Enable verbose output")
//	listCmd := func(ctx context.Context, args []string) error {
//	    if *verbose {
//	        fmt.Println("Verbose listing")
//	    } else {
//	        fmt.Println("Listing")
//	    }
//	    return nil
//	}
//	cmd := cli.NewCommand("list", listCmd, &listFlags, "List resources")
//	cli.Run(context.Background(), []cli.Command{cmd}, os.Args)
//
// Example (Object-based command):
//
//	type GreetCommand struct {
//	    name  string
//	}
//	func (c *GreetCommand) Command() (*flag.FlagSet, cli.CmdFunc) {
//	    fset := flag.NewFlagSet("greet", flag.ContinueOnError)
//	    fset.StringVar(&c.name, "name", "World", "Name to greet")
//	    return fset, func(ctx context.Context, args []string) error {
//	        fmt.Printf("Hello, %s!\n", c.name)
//	        return nil
//	    }
//	}
//	cmd := &GreetCommand{}
//	cli.Run(context.Background(), []cli.Command{cmd}, os.Args)
//
// Optional interfaces for documentation:
//
//	type Purpose interface {
//	  // One line use or summary for the command.
//	  Purpose() string
//	}
//
//	type Description interface {
//	  // Multi line or multi-paragraph help for the command.
//	  Description() string
//	}
package cli

import (
	"context"
	"flag"
	"os"
)

// CmdFunc defines the behavior of a CLI command. It accepts a context for
// cancellation and a slice of arguments (excluding the command name and flags),
// returning an error if execution fails.
//
// Example:
//
//	cmd := func(ctx context.Context, args []string) error {
//	    fmt.Println("Hello, CLI")
//	    return nil
//	}
//	command := cli.NewCommand("hello", cmd, nil, "Print greeting")
type CmdFunc func(ctx context.Context, args []string) error

// Command defines a CLI command or subcommand group. Implementations must
// provide a flag.FlagSet and CmdFunc via the Command method. The FlagSet's name
// serves as the command name and must be non-empty.
//
// Commands may implement optional interfaces for documentation:
//   - Purpose() string: Returns a brief description.
//   - Description() string: Returns detailed help text.
//
// Create commands using NewCommand, NewGroup, or custom types.
//
// Example:
//
//	type VersionCommand struct {
//	    flags flag.FlagSet
//	}
//	func (c *VersionCommand) Command() (*flag.FlagSet, cli.CmdFunc) {
//	    c.flags.Init("version", flag.ContinueOnError)
//	    return c.flags, func(ctx context.Context, args []string) error {
//	        fmt.Println("Version 1.0.0")
//	        return nil
//	    }
//	}
type Command interface {
	// Command returns the command's FlagSet and implementation function.
	// The FlagSet's name is the command name and must be non-empty.
	Command() (*flag.FlagSet, CmdFunc)
}

type basicCmd struct {
	cmd     CmdFunc
	fset    *flag.FlagSet
	purpose string
}

func (v *basicCmd) Command() (*flag.FlagSet, CmdFunc) {
	return v.fset, v.cmd
}

func (v *basicCmd) Purpose() string {
	return v.purpose
}

// NewCommand creates a function-based command with the specified name, function,
// flags, and description. The flag.FlagSet is optional; if nil, no flags are
// supported. The package overrides flag.FlagSet's default error handling.
//
// Example:
//
//	var flags flag.FlagSet
//	name := flags.String("name", "World", "Name to greet")
//	cmd := func(ctx context.Context, args []string) error {
//	    fmt.Printf("Hello, %s!\n", *name)
//	    return nil
//	}
//	command := cli.NewCommand("greet", cmd, &flags, "Greet a user")
func NewCommand(name string, cmd CmdFunc, fset *flag.FlagSet, desc string) Command {
	if fset == nil {
		fset = flag.NewFlagSet(name, flag.ContinueOnError)
	} else {
		fset.Init(name, flag.ContinueOnError)
	}
	return &basicCmd{cmd: cmd, fset: fset, purpose: desc}
}

// NewGroup creates a subcommand group with the specified name, description, and
// subcommands. Returns a Command, enabling nested command hierarchies.
//
// Example:
//
//	startCmd := cli.NewCommand("start", startFunc, nil, "Start server")
//	stopCmd := cli.NewCommand("stop", stopFunc, nil, "Stop server")
//	group := cli.NewGroup("server", "Server operations", startCmd, stopCmd)
func NewGroup(name, helpLine string, cmds ...Command) Command {
	return &groupCmd{
		flags:   flag.NewFlagSet(name, flag.ContinueOnError),
		subcmds: cmds,
		purpose: helpLine,
	}
}

// Run executes the CLI, parsing arguments to invoke a command from the provided
// commands. It supports built-in "help", "flags", and "commands" for
// documentation and uses the context for cancellation. Returns an error if
// parsing or execution fails.
//
// Example:
//
//	cmd := cli.NewCommand("version", versionCmd, nil, "Display version")
//	err := cli.Run(context.Background(), []cli.Command{cmd}, os.Args)
func Run(ctx context.Context, cmds []Command, args []string) error {
	if cmds == nil {
		return os.ErrInvalid
	}
	root := groupCmd{
		flags:   flag.CommandLine,
		subcmds: cmds,
	}
	// If user passes os.Args, turn it into os.Args[1:] instead.
	if &args[0] == &os.Args[0] {
		args = os.Args[1:]
	}
	return root.run(ctx, args)
}
