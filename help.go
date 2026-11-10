// Copyright (c) 2025 Visvasity LLC

package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

func numFlags(fs *flag.FlagSet) int {
	n := 0
	fs.VisitAll(func(*flag.Flag) { n++ })
	return n
}

// printDefaults prints the flag defaults to w without permanently redirecting
// the FlagSet's output.
func printDefaults(w io.Writer, fs *flag.FlagSet) {
	saved := fs.Output()
	fs.SetOutput(w)
	defer fs.SetOutput(saved)
	fs.PrintDefaults()
}

func getUsage(cmdpath []*cmdData) string {
	var words []string

	for i, c := range cmdpath {
		name := c.name
		if i == 0 {
			// Root command name is the program path.
			_, name = filepath.Split(name)
		}
		words = append(words, name)
	}

	for _, c := range cmdpath {
		if n := numFlags(c.fset); n != 0 {
			words = append(words, "<flags>")
			break
		}
	}

	if _, ok := cmdpath[len(cmdpath)-1].cmd.(*groupCmd); ok {
		words = append(words, "<subcommand>")
	}

	words = append(words, "<args>")
	return strings.Join(words, " ")
}

func getHelpDoc(c Command) string {
	if v, ok := c.(interface{ Description() string }); ok {
		return v.Description()
	}
	return getPurpose(c)
}

func getPurpose(c Command) string {
	if v, ok := c.(interface{ Purpose() string }); ok {
		return v.Purpose()
	}
	return ""
}

// mergeFlags returns a new FlagSet with the flags defined by all commands in
// the command path, except the ones that are also defined in the shadow
// FlagSet when it is non-nil, along with the number of flags in it.
func mergeFlags(cmdpath []*cmdData, shadow *flag.FlagSet) (*flag.FlagSet, int) {
	// A flag may be defined multiple times unfortunately, in which case, we pick
	// the closest/deepest flag.Flag to the currently running command.
	flagMap := make(map[string]*flag.Flag)
	for _, c := range cmdpath {
		c.fset.VisitAll(func(f *flag.Flag) { flagMap[f.Name] = f })
	}
	fset := flag.NewFlagSet("temp", flag.ContinueOnError)
	for name, f := range flagMap {
		if shadow != nil && shadow.Lookup(name) != nil {
			continue
		}
		fset.Var(f.Value, f.Name, f.Usage)
		// Var records the current value as the default, so restore the real one.
		fset.Lookup(f.Name).DefValue = f.DefValue
	}
	return fset, numFlags(fset)
}

// getInheritedFlags returns the flags defined by the ancestors of the current
// command. Flags redefined by the current command shadow the inherited ones.
func getInheritedFlags(cmdpath []*cmdData) (*flag.FlagSet, int) {
	n := len(cmdpath) - 1
	return mergeFlags(cmdpath[:n], cmdpath[n].fset)
}

// getAllFlags returns all flags accepted by the current command, which are
// its own flags and the flags inherited from its ancestors.
func getAllFlags(cmdpath []*cmdData) (*flag.FlagSet, int) {
	return mergeFlags(cmdpath, nil)
}

// getSubcommands returns all subcommand names and purpose as a pair.
func getSubcommands(cmdpath []*cmdData) [][2]string {
	// Built-in commands are only available at the top level.
	var spcmds [][2]string
	if len(cmdpath) == 1 {
		spcmds = specialCmds
	}

	var subcmds, groups [][2]string
	if gc, ok := cmdpath[len(cmdpath)-1].cmd.(*groupCmd); ok {
		for _, c := range gc.subcmds {
			n, _, _ := c.Command()
			s := getPurpose(c)
			if _, ok := c.(*groupCmd); ok {
				groups = append(groups, [2]string{n, s})
			} else {
				subcmds = append(subcmds, [2]string{n, s})
			}
		}
	}
	sort.SliceStable(subcmds, func(i, j int) bool {
		return subcmds[i][0] < subcmds[j][0]
	})
	sort.SliceStable(groups, func(i, j int) bool {
		return groups[i][0] < groups[j][0]
	})

	var all [][2]string
	if len(spcmds) > 0 {
		all = append(all, spcmds...)
	}
	if len(subcmds) > 0 {
		if len(all) > 0 {
			all = append(all, [2]string{})
		}
		all = append(all, subcmds...)
	}
	if len(groups) > 0 {
		if len(all) > 0 {
			all = append(all, [2]string{})
		}
		all = append(all, groups...)
	}
	return all
}

// printSubcommands prints the subcommand names with their purpose, one per
// line. Empty entries separate the sections and are printed as blank lines.
func printSubcommands(w io.Writer, subcmds [][2]string) {
	for _, sub := range subcmds {
		switch {
		case len(sub[1]) > 0:
			fmt.Fprintf(w, "\t%-15s  %s\n", sub[0], sub[1])
		case len(sub[0]) > 0:
			fmt.Fprintf(w, "\t%s\n", sub[0])
		default:
			fmt.Fprintln(w)
		}
	}
}

func (gc *groupCmd) printHelp(ctx context.Context, w io.Writer, cmdpath []*cmdData) error {
	last := cmdpath[len(cmdpath)-1]

	usage := getUsage(cmdpath)
	help := getHelpDoc(last.cmd)
	subcmds := getSubcommands(cmdpath)
	flags, nflags := last.fset, numFlags(last.fset)
	iflags, niflags := getInheritedFlags(cmdpath)

	fmt.Fprintf(w, "Usage: %s\n", usage)
	if len(help) > 0 {
		fmt.Fprintln(w)
		// TODO: Format the help into 80 columns?
		fmt.Fprintf(w, "%s\n", strings.TrimSpace(help))
	}
	if len(subcmds) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Subcommands:\n")
		printSubcommands(w, subcmds)
	}
	if nflags > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Flags:\n")
		printDefaults(w, flags)
	}
	if niflags > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Inherited Flags:\n")
		printDefaults(w, iflags)
	}
	return nil
}
