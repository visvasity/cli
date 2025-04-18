// Copyright (c) 2025 Visvasity LLC

//go:build ignore

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/visvasity/cli"
)

type ClientFlags struct {
	Port        int
	Host        string
	APIPath     string
	HTTPTimeout time.Duration
}

func (cf *ClientFlags) SetFlags(fset *flag.FlagSet) {
	fset.IntVar(&cf.Port, "connect-port", 10000, "TCP port number for the api endpoint")
	fset.StringVar(&cf.Host, "connect-host", "127.0.0.1", "Hostname or IP address for the api endpoint")
	fset.StringVar(&cf.APIPath, "api-path", "/", "base path to the api handler")
	fset.DurationVar(&cf.HTTPTimeout, "http-timeout", 30*time.Second, "http client timeout")
}

type DBFlags struct {
	dbURLPath string
}

func (f *DBFlags) SetFlags(fset *flag.FlagSet) {
	fset.StringVar(&f.dbURLPath, "db-url-path", "/db", "path to db api handler")
}

type List struct {
	ClientFlags

	DBFlags

	KeyRe string

	ValueType string

	PrintTemplate string

	InOrder, Descend bool
}

func (c *List) Command() (*flag.FlagSet, cli.CmdFunc) {
	fset := flag.NewFlagSet("list", flag.ContinueOnError)
	c.DBFlags.SetFlags(fset)
	c.ClientFlags.SetFlags(fset)
	fset.StringVar(&c.KeyRe, "key-regexp", "", "regular expression to pick keys")
	fset.StringVar(&c.ValueType, "value-type", "", "gob type name for the values")
	fset.StringVar(&c.PrintTemplate, "print-template", "", "text/template to print the value")
	fset.BoolVar(&c.InOrder, "in-order", false, "when true, prints in ascending order")
	fset.BoolVar(&c.Descend, "descend", false, "when true, prints in descending order")
	return fset, cli.CmdFunc(c.Run)
}

func (c *List) Purpose() string {
	return "Prints keys and values in the database"
}

func (c *List) Run(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("command takes no arguments")
	}

	js, _ := json.MarshalIndent(&c, "", "  ")
	fmt.Printf("%s\n", js)
	return nil
}

func main() {
	cmds := []cli.Command{
		new(List),
	}
	if err := cli.Run(context.Background(), cmds, os.Args); err != nil {
		log.Fatal(err)
	}
}
