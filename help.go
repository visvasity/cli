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

func getName(c Command) string {
	name, _, _ := c.Command()
	_, file := filepath.Split(name)
	return file
}

func getUsage(cmdpath []*cmdData) string {
	var words []string

	for i, c := range cmdpath {
		name := c.fset.Name()
		if i == 0 {
			_, name = filepath.Split(c.fset.Name())
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
	if v, ok := c.(*groupCmd); ok {
		return v.purpose
	}
	return ""
}

func getFlags(c Command) (*flag.FlagSet, int) {
	_, fs, _ := c.Command()
	return fs, numFlags(fs)
}

func getInheritedFlags(cmdpath []*cmdData) (*flag.FlagSet, int) {
	flagMap := make(map[string][]*flag.Flag)
	collector := func(f *flag.Flag) {
		fs := flagMap[f.Name]
		flagMap[f.Name] = append(fs, f)
	}
	// Collect flag.Flag values defined by ancestors from the command path. A
	// flag may be defined multiple times unfortunately, in which case, we pick
	// the closest/deepest flag.Flag to the currently running command.
	for i := 0; i < len(cmdpath)-1; i++ {
		cmdpath[i].fset.VisitAll(collector)
	}
	fset := flag.NewFlagSet("temp", flag.ContinueOnError)
	for _, fs := range flagMap {
		last := fs[len(fs)-1]
		fset.Var(last.Value, last.Name, last.Usage)
	}
	return fset, numFlags(fset)
}

// getSubcommands returns all subcommand names and purpose as a pair.
func getSubcommands(cmdpath []*cmdData) [][2]string {
	var spcmds [][2]string
	if len(cmdpath) == 1 {
		spcmds = [][2]string{
			{"help", "Describe commands and flags"},
			{"flags", "Describe all known flags"},
			{"commands", "Lists all command names"},
		}
	}

	var subcmds, groups [][2]string
	if gc, ok := cmdpath[len(cmdpath)-1].cmd.(*groupCmd); ok {
		for _, c := range gc.subcmds {
			n, s := getName(c), getPurpose(c)
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

func (gc *groupCmd) printHelp(ctx context.Context, w io.Writer, cmdpath []*cmdData) error {
	last := cmdpath[len(cmdpath)-1]

	usage := getUsage(cmdpath)
	help := getHelpDoc(last.cmd)
	subcmds := getSubcommands(cmdpath)
	flags, nflags := getFlags(last.cmd)
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
		for _, sub := range subcmds {
			if len(sub[1]) > 0 {
				fmt.Fprintf(w, "\t%-15s  %s\n", sub[0], sub[1])
			} else if len(sub[0]) > 0 {
				fmt.Fprintf(w, "\t%-15s\n", sub[0])
			} else {
				fmt.Fprintln(w)
			}
		}
	}
	if nflags > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Flags:\n")
		flags.SetOutput(w)
		flags.PrintDefaults()
	}
	if niflags > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Inherited Flags:\n")
		iflags.SetOutput(w)
		iflags.PrintDefaults()
	}
	return nil
}
