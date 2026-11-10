// Copyright (c) 2025 Visvasity LLC

package cli

import (
	"context"
	"flag"
	"os"
	"strings"
	"testing"
)

type unnamedFlagSetCmd struct {
	name string
}

func (c *unnamedFlagSetCmd) Command() (string, *flag.FlagSet, CmdFunc) {
	fset := new(flag.FlagSet)
	fset.StringVar(&c.name, "name", "World", "Name to greet")
	return "greet", fset, func(context.Context, []string) error { return nil }
}

type nilFlagSetCmd struct {
	args []string
}

func (c *nilFlagSetCmd) Command() (string, *flag.FlagSet, CmdFunc) {
	return "noflags", nil, func(_ context.Context, args []string) error {
		c.args = args
		return nil
	}
}

func noop(context.Context, []string) error {
	return nil
}

// runCapture runs the commands and returns everything written to the stdout.
func runCapture(t *testing.T, cmds []Command, args ...string) string {
	t.Helper()
	var sb strings.Builder
	if err := Run(WithStdout(context.Background(), &sb), cmds, args); err != nil {
		t.Fatal(err)
	}
	return sb.String()
}

func TestHelpHonorsStdout(t *testing.T) {
	cmds := []Command{NewCommand("version", noop, nil, "Print version")}
	for _, special := range []string{"help", "flags", "commands"} {
		if out := runCapture(t, cmds, special); len(out) == 0 {
			t.Errorf("%s: want output in the redirected stdout, got none", special)
		}
	}
	// Command groups print help when no subcommand is selected.
	var sb strings.Builder
	if err := Run(WithStdout(context.Background(), &sb), cmds, nil); err == nil {
		t.Errorf("want error when no command is selected, got nil")
	}
	if out := sb.String(); !strings.Contains(out, "version") {
		t.Errorf("want help with version subcommand, got %q", out)
	}
}

func TestUsageWithUnnamedFlagSet(t *testing.T) {
	out := runCapture(t, []Command{new(unnamedFlagSetCmd)}, "greet", "-h")
	usage, _, _ := strings.Cut(out, "\n")
	if !strings.Contains(usage, " greet ") {
		t.Errorf("want command name in the usage line, got %q", usage)
	}
}

func TestCommandsOutput(t *testing.T) {
	cmds := []Command{
		NewGroup("server", "Server operations", NewCommand("start", noop, nil, "")),
		NewCommand("version", noop, nil, "Print version"),
	}
	out := runCapture(t, cmds, "commands", "server")
	if want := "\tstart\n"; out != want {
		t.Errorf("want %q, got %q", want, out)
	}
	out = runCapture(t, cmds, "commands")
	for _, line := range strings.Split(out, "\n") {
		if len(line) > 0 && len(strings.TrimSpace(line)) == 0 {
			t.Errorf("want empty separator lines, got %q in %q", line, out)
		}
	}
}

func TestInheritedFlags(t *testing.T) {
	startFlags := new(flag.FlagSet)
	startFlags.String("format", "text", "start output format")
	group := NewGroup("server", "Server operations", NewCommand("start", noop, startFlags, "Start"))
	_, groupFlags, _ := group.Command()
	groupFlags.Int("port", 8080, "server port")
	groupFlags.String("format", "json", "server output format")

	out := runCapture(t, []Command{group}, "server", "-port", "9090", "start", "-h")
	_, inherited, ok := strings.Cut(out, "Inherited Flags:\n")
	if !ok {
		t.Fatalf("want inherited flags section, got %q", out)
	}
	if !strings.Contains(inherited, "(default 8080)") {
		t.Errorf("want the real default for the inherited port flag, got %q", inherited)
	}
	if strings.Contains(inherited, "server output format") {
		t.Errorf("want shadowed format flag to be skipped, got %q", inherited)
	}
}

func TestFlagsIncludesInheritedFlags(t *testing.T) {
	startFlags := new(flag.FlagSet)
	startFlags.String("format", "text", "start output format")
	group := NewGroup("server", "Server operations", NewCommand("start", noop, startFlags, "Start"))
	_, groupFlags, _ := group.Command()
	groupFlags.Int("port", 8080, "server port")
	groupFlags.String("format", "json", "server output format")

	out := runCapture(t, []Command{group}, "flags", "server", "-port", "9090", "start")
	if !strings.Contains(out, "start output format") {
		t.Errorf("want the flags of the command, got %q", out)
	}
	if !strings.Contains(out, "server port (default 8080)") {
		t.Errorf("want the inherited port flag with its real default, got %q", out)
	}
	if strings.Contains(out, "server output format") {
		t.Errorf("want shadowed format flag to be skipped, got %q", out)
	}

	// Command group has no ancestors with the start command's flags.
	out = runCapture(t, []Command{group}, "flags", "server")
	if !strings.Contains(out, "server port") || !strings.Contains(out, "server output format") {
		t.Errorf("want the flags of the group, got %q", out)
	}
	if strings.Contains(out, "start output format") {
		t.Errorf("want no flags from the subcommands, got %q", out)
	}
}

func TestHelpRestoresFlagSetOutput(t *testing.T) {
	fset := new(flag.FlagSet)
	fset.Bool("force", false, "Force stop")
	cmds := []Command{NewCommand("stop", noop, fset, "Stop")}
	for _, args := range [][]string{{"help", "stop"}, {"flags", "stop"}} {
		runCapture(t, cmds, args...)
		if fset.Output() != os.Stderr {
			t.Errorf("%v: want flagset output to be restored", args)
		}
	}
}

func TestNilFlagSet(t *testing.T) {
	cmd := new(nilFlagSetCmd)
	if err := Run(context.Background(), []Command{cmd}, []string{"noflags", "arg"}); err != nil {
		t.Fatal(err)
	}
	if len(cmd.args) != 1 || cmd.args[0] != "arg" {
		t.Fatalf("want [arg], got %v", cmd.args)
	}
	if err := Run(context.Background(), []Command{cmd}, []string{"noflags", "-undefined"}); err == nil {
		t.Fatalf("want error for an undefined flag, got nil")
	}
	runCapture(t, []Command{cmd}, "help", "noflags")
}
