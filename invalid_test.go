// Copyright (c) 2025 Visvasity LLC

package cli

import (
	"context"
	"flag"
	"strings"
	"testing"
)

type emptyNameCmd struct{}

func (c *emptyNameCmd) Command() (string, *flag.FlagSet, CmdFunc) {
	return "", nil, noop
}

func TestInvalidCommands(t *testing.T) {
	start := NewCommand("start", noop, nil, "Start server")
	stop := NewCommand("stop", noop, nil, "Stop server")
	version := NewCommand("version", noop, nil, "Print version")
	server := NewGroup("server", "Server operations", start, stop)

	tests := []struct {
		name    string
		cmds    []Command
		args    []string
		wantErr string
	}{
		{
			name:    "Nil command",
			cmds:    []Command{version, NewCommand("", noop, nil, "")},
			args:    []string{"version"},
			wantErr: "command is nil: #2",
		},
		{
			name:    "Nil group",
			cmds:    []Command{NewGroup("", "", start), version},
			args:    []string{"version"},
			wantErr: "command is nil: #1",
		},
		{
			name:    "Nil command in a group",
			cmds:    []Command{NewGroup("server", "", start, nil)},
			args:    []string{"server", "start"},
			wantErr: "server: command is nil: #2",
		},
		{
			name:    "Duplicate command",
			cmds:    []Command{version, server, NewCommand("version", noop, nil, "")},
			args:    []string{"version"},
			wantErr: "command defined multiple times: version",
		},
		{
			name:    "Duplicate command and group",
			cmds:    []Command{server, NewCommand("server", noop, nil, "")},
			args:    []string{"help"},
			wantErr: "command defined multiple times: server",
		},
		{
			name:    "Duplicate command in a group",
			cmds:    []Command{NewGroup("server", "", start, stop, start)},
			args:    []string{"server", "stop"},
			wantErr: "server: command defined multiple times: start",
		},
		{
			name:    "Nil command in an unused group",
			cmds:    []Command{version, NewGroup("server", "", start, nil)},
			args:    []string{"version"},
			wantErr: "server: command is nil: #2",
		},
		{
			name:    "Duplicate command in an unused group",
			cmds:    []Command{version, NewGroup("server", "", start, stop, start)},
			args:    []string{"help"},
			wantErr: "server: command defined multiple times: start",
		},
		{
			name:    "Duplicate command in an unused nested group",
			cmds:    []Command{version, NewGroup("db", "", NewGroup("backup", "", start, start))},
			args:    []string{"version"},
			wantErr: "db: backup: command defined multiple times: start",
		},
		{
			name:    "Empty command name",
			cmds:    []Command{version, new(emptyNameCmd)},
			args:    []string{"version"},
			wantErr: "command name is empty: #2",
		},
		{
			name:    "Empty command name in an unused group",
			cmds:    []Command{version, NewGroup("server", "", new(emptyNameCmd))},
			args:    []string{"version"},
			wantErr: "server: command name is empty: #1",
		},
		{
			name:    "Group with no subcommands",
			cmds:    []Command{version, NewGroup("server", "Server operations")},
			args:    []string{"version"},
			wantErr: "command group has no subcommands: server",
		},
		{
			name:    "Nested group with no subcommands",
			cmds:    []Command{version, NewGroup("db", "", start, NewGroup("backup", ""))},
			args:    []string{"db", "start"},
			wantErr: "db: command group has no subcommands: backup",
		},
		{
			name: "Same command name in different groups",
			cmds: []Command{server, NewGroup("client", "", start), start},
			args: []string{"client", "start"},
		},
		{
			name:    "No command",
			cmds:    []Command{server, version},
			wantErr: "command not specified",
		},
		{
			name:    "No subcommand",
			cmds:    []Command{server, version},
			args:    []string{"server"},
			wantErr: "subcommand not specified: server",
		},
		{
			name: "Help for a group",
			cmds: []Command{server, version},
			args: []string{"server", "-h"},
		},
		{
			name: "Help command for a group",
			cmds: []Command{server, version},
			args: []string{"help", "server"},
		},
		{
			name: "Help with no command",
			cmds: []Command{server, version},
			args: []string{"help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			err := Run(WithStdout(context.Background(), &sb), tt.cmds, tt.args)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Run: got error %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("Run: got error %v, want %q", err, tt.wantErr)
			}
		})
	}

	// Help is still printed when a command group is run without a subcommand.
	var sb strings.Builder
	if err := Run(WithStdout(context.Background(), &sb), []Command{server}, []string{"server"}); err == nil {
		t.Errorf("Run: got nil, want error")
	}
	if out := sb.String(); !strings.Contains(out, "Subcommands:") || !strings.Contains(out, "start") {
		t.Errorf("want help with the subcommands, got %q", out)
	}
}
