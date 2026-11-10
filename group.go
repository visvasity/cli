// Copyright (c) 2025 Visvasity LLC

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"
)

type groupCmd struct {
	flags      *flag.FlagSet
	subcmds    []Command
	specialCmd string
	purpose    string
}

// NewGroup creates a subcommand group with the specified name, purpose, and
// subcommands. Returns a Command, enabling nested command hierarchies. Returns
// nil if group name is empty, which is reported as an error by Run.
//
// Example:
//
//	startCmd := cli.NewCommand("start", startFunc, nil, "Start server")
//	stopCmd := cli.NewCommand("stop", stopFunc, nil, "Stop server")
//	group := cli.NewGroup("server", "Server operations", startCmd, stopCmd)
func NewGroup(name, purpose string, cmds ...Command) Command {
	if len(name) == 0 {
		return nil
	}
	return &groupCmd{
		flags:   flag.NewFlagSet(name, flag.ContinueOnError),
		subcmds: cmds,
		purpose: purpose,
	}
}

// specialCmds holds the built-in command names and their purpose as a pair.
var specialCmds = [][2]string{
	{"help", "Describe commands and flags"},
	{"flags", "Describe all known flags"},
	{"commands", "Lists all command names"},
}

func isSpecialCmd(name string) bool {
	return slices.ContainsFunc(specialCmds, func(c [2]string) bool { return c[0] == name })
}

// Command implements Command interface.
func (gc *groupCmd) Command() (string, *flag.FlagSet, CmdFunc) {
	return gc.flags.Name(), gc.flags, nil
}

// Purpose returns the single line description of the group.
func (gc *groupCmd) Purpose() string {
	return gc.purpose
}

func (gc *groupCmd) printFlags(ctx context.Context, w io.Writer, cmdpath []*cmdData) error {
	// Flags of the ancestors are also accepted by the command.
	flags, _ := getAllFlags(cmdpath)
	printDefaults(w, flags)
	return nil
}

func (gc *groupCmd) printCommands(ctx context.Context, w io.Writer, cmdpath []*cmdData) error {
	printSubcommands(w, getSubcommands(cmdpath))
	return nil
}

type cmdData struct {
	name string
	fset *flag.FlagSet
	fun  CmdFunc
	cmd  Command

	// subcmds holds the subcommands by their name for the command groups.
	subcmds map[string]*cmdData
}

// loadCommands collects the command data of all input commands by their name,
// which includes the subcommands of the command groups recursively. Returns an
// error if any command is nil, has an empty name, shares its name with another
// command in the same group or is a command group with no subcommands.
func loadCommands(cmds []Command) (map[string]*cmdData, error) {
	m := make(map[string]*cmdData)
	for i, c := range cmds {
		// NewCommand and NewGroup return nil when the name is empty.
		if c == nil {
			return nil, fmt.Errorf("command is nil: #%d", i+1)
		}
		name, fs, fn := c.Command()
		if len(name) == 0 {
			return nil, fmt.Errorf("command name is empty: #%d", i+1)
		}
		if _, ok := m[name]; ok {
			return nil, fmt.Errorf("command defined multiple times: %s", name)
		}
		// Like in NewCommand, a nil FlagSet means command has no flags.
		if fs == nil {
			fs = flag.NewFlagSet(name, flag.ContinueOnError)
		}
		data := &cmdData{
			name: name,
			fset: fs,
			fun:  fn,
			cmd:  c,
		}
		if sg, ok := c.(*groupCmd); ok {
			if len(sg.subcmds) == 0 {
				return nil, fmt.Errorf("command group has no subcommands: %s", name)
			}
			subcmds, err := loadCommands(sg.subcmds)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			data.subcmds = subcmds
		}
		m[name] = data
	}
	return m, nil
}

func (gc *groupCmd) resolve(ctx context.Context, args []string) ([]*cmdData, []string, error) {
	type boolFlag interface {
		flag.Value
		IsBoolFlag() bool
	}

	// All commands are checked up front, so that an invalid command is reported
	// irrespective of the command selected by the arguments.
	cmdDataMap, err := loadCommands(gc.subcmds)
	if err != nil {
		return nil, nil, err
	}

	cmdpath := []*cmdData{
		{
			name: gc.flags.Name(),
			fset: gc.flags,
			cmd:  gc,
		},
	}

	lookup := func(s string) (*flag.Flag, bool) {
		for i := len(cmdpath) - 1; i >= 0; i-- {
			if f := cmdpath[i].fset.Lookup(s); f != nil {
				return f, true
			}
		}
		return nil, false
	}

	var i int
	for i = 0; i < len(args); i++ {
		s := args[i]

		// stop resolving subcmds and flags
		if s == "--" {
			i++
			break
		}

		// Non-flag argument
		if len(s) < 2 || s[0] != '-' {
			// non-flag argument to the last subcmd
			if len(cmdDataMap) == 0 {
				break
			}

			subcmd, ok := cmdDataMap[s]
			if !ok {
				// handle one of special commands: help, flags, commands
				if len(cmdpath) == 1 && isSpecialCmd(s) {
					gc.specialCmd = s
					continue
				}
				return nil, nil, fmt.Errorf("command not defined: %s", s)
			}
			cmdpath = append(cmdpath, subcmd)

			// handle subcommands from a command group; non-group commands have no
			// subcommands, which stops the subcommand processing, but continues to
			// resolve flags
			cmdDataMap = subcmd.subcmds
			continue
		}

		// remove the '-' or '--' prefix and '=...' suffix
		name := s[1:]
		if s[1] == '-' {
			name = s[2:]
		}
		if len(name) == 0 || name[0] == '-' || name[0] == '=' {
			return nil, nil, fmt.Errorf("bad flag syntax: %s", s)
		}
		name, value, hasValue := strings.Cut(name, "=")

		// check for the flag in all the parent FlagSets
		flag, ok := lookup(name)
		if !ok {
			if name == "help" || name == "h" {
				gc.specialCmd = "help"
				continue
			}
			return nil, nil, fmt.Errorf("flag provided but not defined: -%s", name)
		}

		// handle boolean flag, which doesn't need an argument.
		if fv, ok := flag.Value.(boolFlag); ok && fv.IsBoolFlag() {
			if hasValue {
				if err := fv.Set(value); err != nil {
					return nil, nil, fmt.Errorf("invalid boolean value %q for -%s: %w", value, name, err)
				}
			} else {
				if err := fv.Set("true"); err != nil {
					return nil, nil, fmt.Errorf("invalid boolean flag %s: %w", name, err)
				}
			}
			continue
		}

		// non-boolean flags must have a value, which might be the next argument.
		if !hasValue && i+1 < len(args) {
			hasValue = true
			value = args[i+1]
			i++
		}
		if !hasValue {
			return nil, nil, fmt.Errorf("flag needs an argument: -%s", name)
		}
		if err := flag.Value.Set(value); err != nil {
			return nil, nil, fmt.Errorf("invalid value %q for flag -%s: %w", value, name, err)
		}
	}

	return cmdpath, args[i:], nil
}

func (gc *groupCmd) run(ctx context.Context, args []string) error {
	cmdpath, args, err := gc.resolve(ctx, args)
	if err != nil {
		return err
	}

	// Built-in commands also honor the stdout redirection.
	w := Stdout(ctx)
	switch gc.specialCmd {
	case "help":
		return gc.printHelp(ctx, w, cmdpath)
	case "flags":
		return gc.printFlags(ctx, w, cmdpath)
	case "commands":
		return gc.printCommands(ctx, w, cmdpath)
	}

	last := cmdpath[len(cmdpath)-1]
	fun := last.fun
	if fun == nil {
		if err := gc.printHelp(ctx, w, cmdpath); err != nil {
			return err
		}
		// A command group cannot run without a subcommand.
		if _, ok := last.cmd.(*groupCmd); ok {
			if len(cmdpath) == 1 {
				return errors.New("command not specified")
			}
			return fmt.Errorf("subcommand not specified: %s", last.name)
		}
		return nil
	}

	return fun(ctx, args)
}
