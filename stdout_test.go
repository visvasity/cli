// Copyright (c) 2025 Visvasity LLC

package cli_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/visvasity/cli"
)

func TestStdoutCapture(t *testing.T) {
	ctx := context.Background()

	helloCmd := func(ctx context.Context, _ []string) error {
		stdout := cli.Stdout(ctx)
		fmt.Fprintf(stdout, "hello world")
		return nil
	}

	var sb strings.Builder
	ctx = cli.WithStdout(ctx, &sb)

	cmd := cli.NewCommand("hello", helloCmd, nil, "Prints hello world")
	if err := cli.Run(ctx, []cli.Command{cmd}, []string{"hello"}); err != nil {
		t.Fatal(err)
	}

	if s := sb.String(); s != "hello world" {
		t.Fatalf("want 'hello world', got %q", s)
	}
}
