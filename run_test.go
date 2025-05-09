// Copyright (c) 2025 Visvasity LLC

package cli

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
)

type TestCmd struct {
	name  string
	flags *flag.FlagSet
	args  []string
}

func newTestCmd(name string) *TestCmd {
	return &TestCmd{
		name:  name,
		flags: flag.NewFlagSet(name, flag.ContinueOnError),
	}
}

func (t *TestCmd) Command() (string, *flag.FlagSet, CmdFunc) {
	return t.name, t.flags, CmdFunc(func(_ context.Context, args []string) error {
		log.Println("running", t.name, "with args", args)
		t.args = args
		return nil
	})
}

func printVersion(context.Context, []string) error {
	fmt.Fprintln(os.Stderr, "...")
	return nil
}

func TestRun(t *testing.T) {
	ctx := context.Background()

	run := newTestCmd("run")
	background := run.flags.Bool("background", false, "set to run in background")

	jobsList := newTestCmd("list")
	jobsList.flags.String("format", "json", "list output format")
	jobsSummary := newTestCmd("summary")
	jobsSummary.flags.String("format", "json", "summary output format")
	jobs := NewGroup("jobs", "manage jobs", jobsList, jobsSummary)

	jobPause := newTestCmd("pause")
	jobPause.flags.Duration("timeout", 0, "pause duration")
	jobResume := newTestCmd("resume")
	jobResume.flags.Duration("timeout", 0, "resume duration")
	jobCancel := newTestCmd("cancel")
	jobCancel.flags.Duration("after", 0, "cancellation delay")
	jobArchive := newTestCmd("archive")
	jobDelete := newTestCmd("delete")
	job := NewGroup("job", "manage single job", jobPause, jobResume, jobCancel, jobArchive, jobDelete)

	dbGet := newTestCmd("get")
	dbSet := newTestCmd("set")
	dbDelete := newTestCmd("delete")
	dbScan := newTestCmd("scan")
	dbBackup := newTestCmd("backup")
	db := NewGroup("db", "manage database", dbGet, dbSet, dbDelete, dbScan, dbBackup)

	cmds := []Command{run, jobs, job, db, NewCommand("version", printVersion, nil, "print version")}

	{
		args := []string{"db", "scan", "db-scan-argument"}
		if err := Run(ctx, cmds, args); err != nil {
			t.Fatal(err)
		}
		if len(dbScan.args) != 1 || dbScan.args[0] != "db-scan-argument" {
			t.Fatalf("want `db-scan-argument`, got %v", dbScan.args)
		}
	}

	{
		args := []string{"run", "-background", "run-argument"}
		if err := Run(ctx, cmds, args); err != nil {
			t.Fatal(err)
		}
		if len(run.args) != 1 || run.args[0] != "run-argument" {
			t.Fatalf("want `run-argument`, got %v", run.args)
		}
		if *background == false {
			t.Fatalf("want true, got false")
		}
	}
}
