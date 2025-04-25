// Copyright (c) 2025 Visvasity LLC

package cli

import (
	"context"
	"flag"
	"log"
	"testing"
)

type TestKeyCmd struct {
	key string
}

func (t *TestKeyCmd) Command() (string, *flag.FlagSet, CmdFunc) {
	fset := new(flag.FlagSet)
	fset.StringVar(&t.key, "key", "initial", "Name of the key")
	return "test-object", fset, t.run
}

func (t *TestKeyCmd) run(ctx context.Context, args []string) error {
	log.Printf("key=%s", t.key)
	return nil
}

func TestMultipleRuns(t *testing.T) {
	ctx := context.Background()

	var port int
	fset := new(flag.FlagSet)
	fset.IntVar(&port, "port", 10000, "TCP Port number")
	cmdPrint := func(ctx context.Context, args []string) error {
		log.Printf("port=%d", port)
		return nil
	}
	cmd1 := NewCommand("test-func", cmdPrint, fset, "Prints flags")
	cmd2 := new(TestKeyCmd)
	if err := Run(ctx, []Command{cmd1, cmd2}, []string{"test-func", "-port", "9999"}); err != nil {
		t.Fatal(err)
	}
	if err := Run(ctx, []Command{cmd1, cmd2}, []string{"test-func", "-port", "1111"}); err != nil {
		t.Fatal(err)
	}
	if err := Run(ctx, []Command{cmd1, cmd2}, []string{"test-object", "-key", "key1"}); err != nil {
		t.Fatal(err)
	}
	if err := Run(ctx, []Command{cmd1, cmd2}, []string{"test-object", "-key", "key2"}); err != nil {
		t.Fatal(err)
	}
}
