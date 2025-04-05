// Copyright (c) 2025 Visvasity LLC

//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/visvasity/cli"
)

func printVersion(context.Context, []string) error {
	fmt.Fprintln(os.Stdout, "version 1.0.0")
	return nil
}

func main() {
	cmds := []cli.Command{
		cli.NewCommand("version", printVersion, nil, "print version information"),
	}
	if err := cli.Run(context.Background(), cmds, os.Args); err != nil {
		log.Fatal(err)
	}
}
