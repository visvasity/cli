// Copyright (c) 2025 Visvasity LLC

package cli_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/visvasity/cli"
)

func TestPackageExample1(t *testing.T) {
	var listFlags flag.FlagSet
	verbose := listFlags.Bool("v", false, "Enable verbose output")
	listCmd := func(ctx context.Context, args []string) error {
		if *verbose {
			fmt.Println("Verbose listing")
		} else {
			fmt.Println("Listing")
		}
		return nil
	}
	cmd := cli.NewCommand("list", listCmd, &listFlags, "List resources")
	cli.Run(context.Background(), []cli.Command{cmd}, os.Args)
}

type GreetCommand struct {
	name string
}

func (c *GreetCommand) Command() (*flag.FlagSet, cli.CmdFunc) {
	fset := flag.NewFlagSet("greet", flag.ContinueOnError)
	fset.StringVar(&c.name, "name", "World", "Name to greet")
	return fset, func(ctx context.Context, args []string) error {
		fmt.Printf("Hello, %s!\n", c.name)
		return nil
	}
}

func TestPackageExample2(t *testing.T) {
	cmd := &GreetCommand{}
	cli.Run(context.Background(), []cli.Command{cmd}, os.Args)
}
