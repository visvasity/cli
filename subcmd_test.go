// Copyright (c) 2025 Visvasity LLC

package cli

import (
	"context"
	"flag"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// TestSubcommandExecution verifies that the correct subcommand is executed
// when commands are organized as subcommands within a group.
func TestSubcommandExecution(t *testing.T) {
	// Test context
	ctx := context.Background()

	// Create subcommands
	startFset := flag.NewFlagSet("start", flag.ContinueOnError)
	startPort := startFset.Int("port", 8080, "Server port")
	var startArgs []string
	startCmd := NewCommand("start", func(ctx context.Context, args []string) error {
		startArgs = args
		if *startPort != 8080 {
			return fmt.Errorf("port %d", *startPort)
		}
		return nil
	}, startFset, "Start the server")

	stopFset := flag.NewFlagSet("stop", flag.ContinueOnError)
	force := stopFset.Bool("force", false, "Force stop")
	var stopArgs []string
	stopCmd := NewCommand("stop", func(ctx context.Context, args []string) error {
		stopArgs = args
		if *force {
			return fmt.Errorf("forced stop")
		}
		return nil
	}, stopFset, "Stop the server")

	// Create a group
	serverGroup := NewGroup("server", "Server operations", startCmd, stopCmd)

	// Test cases
	tests := []struct {
		name     string
		args     []string
		wantCmd  string
		wantArgs []string
		wantErr  string
	}{
		{
			name:     "Run start subcommand",
			args:     []string{"server", "start", "arg1"},
			wantCmd:  "start",
			wantArgs: []string{"arg1"},
			wantErr:  "",
		},
		{
			name:     "Run start with port flag",
			args:     []string{"server", "start", "-port=9090"},
			wantCmd:  "start",
			wantArgs: []string{},
			wantErr:  "port 9090",
		},
		{
			name:     "Run stop subcommand with force",
			args:     []string{"server", "stop", "-force"},
			wantCmd:  "stop",
			wantArgs: []string{},
			wantErr:  "forced stop",
		},
		{
			name:     "Run invalid subcommand",
			args:     []string{"server", "restart"},
			wantCmd:  "",
			wantArgs: nil,
			wantErr:  "command not defined: restart",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset captured args
			startArgs = nil
			stopArgs = nil

			// Run the command
			err := Run(ctx, []Command{serverGroup}, tt.args)

			// Check error
			if tt.wantErr == "" && err != nil {
				t.Errorf("Run: got error %v, want nil", err)
			} else if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Errorf("Run: got error %v, want error containing %q", err, tt.wantErr)
			}

			// Check which command ran and its args
			var gotArgs []string
			switch tt.wantCmd {
			case "start":
				gotArgs = startArgs
			case "stop":
				gotArgs = stopArgs
			case "":
				if startArgs != nil || stopArgs != nil {
					t.Errorf("Run: unexpected command executed, startArgs=%v, stopArgs=%v", startArgs, stopArgs)
				}
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("Run: got args %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}
