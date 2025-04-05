// Copyright (c) 2025 Visvasity LLC

// Package cli implements minimalistic command-line parsing.
//
// Commands can be defined as functions and objects. Commands can be grouped
// into subcommands for better organization.
//
// Command-line flags can be defined using the standard library's
// flag.FlagSets. Error handling mechanism in the FlagSets is ignored.
//
// # SPECIAL COMMANDS
//
// Special top-level commands "help", "flags", and "commands" are added for
// documentation.
//
// # OPTIONAL INTERFACES
//
// Documentation is collected through optional interfaces. Commands can
// implement `interface{ Synopsis() string }` to provide a short, one-line
// description and `interface{ CommandHelp() string }` to provide a more
// detailed multi-line, multi-paragraph documentation.
//
// # EXAMPLE COMMAND FUNCTION
//
//		func printVersion(ctx context.Context, args []string) error {
//		  fmt.Fprintln(os.Stderr, "...")
//		  return nil
//		}
//
//		func main() {
//		  cmds := []cli.Command{
//		    cli.NewCommand("version", cli.CmdFunc(printVersion), nil, "output version information"),
//		    ...
//		  }
//		  if err := cli.Run(context.Background(), cmds, os.Args); err != nil {
//	      log.Fatal(err)
//	    }
//		}
//
// # EXAMPLE COMMAND OBJECT
//
//		type runCmd struct {
//			background  bool
//			port        int
//			ip          string
//			secretsPath string
//			dataDir     string
//		}
//
//		func (r *runCmd) Run(ctx context.Context, args []string) error {
//			if len(p.dataDir) == 0 {
//				p.dataDir = filepath.Join(os.Getenv("HOME"), ".data")
//			}
//			...
//			return nil
//		}
//
//		func (r *runCmd) Command() (*flag.FlagSet, CmdFunc) {
//			fset := flag.NewFlagSet("run", flag.ContinueOnError)
//			fset.BoolVar(&p.background, "background", false, "runs the daemon in background")
//			fset.IntVar(&p.port, "port", 10000, "TCP port number for the daemon")
//			fset.StringVar(&p.ip, "ip", "0.0.0.0", "TCP ip address for the daemon")
//			fset.StringVar(&p.secretsPath, "secrets-file", "", "path to credentials file")
//			fset.StringVar(&p.dataDir, "data-dir", "", "path to the data directory")
//	    return fset, CmdFunc(r.Run)
//		}
package cli

import (
	"context"
	"flag"
	"os"
)

// CmdFunc defines the signature for command implementation functions.
type CmdFunc func(ctx context.Context, args []string) error

// Command interface defines the requirements for Command objects.
type Command interface {
	// Command returns the command-line flags and command implementation
	// function. Returned FlagSet name is used as the Command name, so it must be
	// non-nil.
	Command() (*flag.FlagSet, CmdFunc)
}

type basicCommand struct {
	cmd      CmdFunc
	fset     *flag.FlagSet
	synopsis string
}

func (v *basicCommand) Command() (*flag.FlagSet, CmdFunc) {
	return v.fset, v.cmd
}

func (v *basicCommand) Synopsis() string {
	return v.synopsis
}

// NewCommand creates a command instance from the input parameters.
func NewCommand(name string, cmd CmdFunc, fset *flag.FlagSet, desc string) Command {
	if fset == nil {
		fset = flag.NewFlagSet(name, flag.ContinueOnError)
	} else {
		fset.Init(name, flag.ContinueOnError)
	}
	return &basicCommand{cmd: cmd, fset: fset, synopsis: desc}
}

// NewGroup makes the input commands into subcommands of a new command with the
// given name and description. This mechanism allows for defining command
// hierarchies of arbitrary depths.
func NewGroup(name, description string, cmds ...Command) Command {
	return &cmdGroup{
		flags:    flag.NewFlagSet(name, flag.ContinueOnError),
		subcmds:  cmds,
		synopsis: description,
	}
}

// Run parses command-line arguments from `args` into flags and subcommands and
// picks the best command to execute from `cmds`. Top-level command flags from
// flag.CommandLine flags are also processed on the way to resolving the best
// command.
func Run(ctx context.Context, cmds []Command, args []string) error {
	if cmds == nil {
		return os.ErrInvalid
	}
	root := cmdGroup{
		flags:   flag.CommandLine,
		subcmds: cmds,
	}
	// If user passes os.Args, turn it into os.Args[1:] instead.
	if &args[0] == &os.Args[0] {
		args = os.Args[1:]
	}
	return root.run(ctx, args)
}
